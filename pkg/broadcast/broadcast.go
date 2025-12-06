package broadcast

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Event is the simple envelope we publish for Balto events.
type Event struct {
	Version   int            `json:"v"`
	Type      string         `json:"type"`
	BackendID string         `json:"backendID,omitempty"`
	Healthy   *bool          `json:"healthy,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
}

type Broadcaster struct {
	client *redis.Client
	stream string
	group  string

	mu       sync.RWMutex
	lastID   string
	consumer string

	handlers []func(Event)

	stop      chan struct{}
	stoppedWg sync.WaitGroup
	persistWg sync.WaitGroup

	pollBlockMs   int64
	trimMaxLen    int64
	persistPeriod time.Duration
	handlerLimit  int
}

func New(client *redis.Client, stream, group string) (*Broadcaster, error) {
	b := &Broadcaster{
		client:        client,
		stream:        stream,
		group:         group,
		lastID:        "0-0", // Default start ID for initial group creation
		consumer:      "node-" + uuid.NewString(),
		stop:          make(chan struct{}),
		pollBlockMs:   500,
		trimMaxLen:    10000,
		persistPeriod: 2 * time.Second,
		handlerLimit:  200,
	}

	ctx := context.Background()
	if err := client.XGroupCreateMkStream(ctx, stream, group, "0").Err(); err != nil {
		if !redis.HasErrorPrefix(err, "BUSYGROUP") {
			return nil, fmt.Errorf("create group: %w", err)
		}
	}

	if s := client.Get(ctx, "balto:lastid:"+group).Val(); s != "" {
		b.lastID = s
	}

	b.start()
	return b, nil
}

// Publish writes an event to the Redis stream, using MaxLenApprox to prevent unbounded growth.
func (b *Broadcaster) Publish(ctx context.Context, ev Event) error {
	if ev.Version == 0 {
		ev.Version = 1
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: b.stream,
		Values: map[string]interface{}{"data": string(data)},
		MaxLen: b.trimMaxLen,
	}).Result()
	return err
}

func (b *Broadcaster) Subscribe(handler func(Event)) {
	b.mu.Lock()
	b.handlers = append(b.handlers, handler)
	b.mu.Unlock()
}

func (b *Broadcaster) start() {
	b.stoppedWg.Add(1)
	go func() {
		defer b.stoppedWg.Done()
		b.pollLoop()
	}()

	b.persistWg.Add(1)
	go func() {
		defer b.persistWg.Done()
		b.persistLoop()
	}()
}

func (b *Broadcaster) pollLoop() {
	for {
		select {
		case <-b.stop:
			return
		default:
		}

		b.mu.RLock()
		consumer := b.consumer
		b.mu.RUnlock()

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.pollBlockMs)*time.Millisecond+100*time.Millisecond)
		streams, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    b.group,
			Consumer: consumer,
			Streams:  []string{b.stream, ">"},
			Count:    100,
			Block:    time.Duration(b.pollBlockMs),
			NoAck:    true,
		}).Result()
		cancel()

		if err != nil {
			if err == redis.Nil { // No new messages after block timeout
				continue
			}
			select {
			case <-time.After(200 * time.Millisecond):
			case <-b.stop:
				return
			}
			continue
		}

		for _, s := range streams {
			for _, msg := range s.Messages {
				raw, ok := msg.Values["data"].(string)
				if !ok {
					continue
				}
				var ev Event
				if err := json.Unmarshal([]byte(raw), &ev); err == nil {
					b.dispatchSafe(ev)
				}

				b.mu.Lock()
				if msg.ID > b.lastID {
					b.lastID = msg.ID
				}
				b.mu.Unlock()
			}
		}
	}
}

func (b *Broadcaster) dispatchSafe(ev Event) {
	b.mu.RLock()
	handlers := append([]func(Event){}, b.handlers...) // Shallow copy to prevent deadlocks/changes during dispatch
	limit := b.handlerLimit
	b.mu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	sem := make(chan struct{}, limit)

	for _, h := range handlers {
		sem <- struct{}{}
		go func(fn func(Event)) {
			defer func() { <-sem }()
			defer func() { _ = recover() }()
			fn(ev)
		}(h)
	}

	for i := 0; i < cap(sem); i++ {
		if i >= len(handlers) {
			break
		}
		sem <- struct{}{}
	}
}

func (b *Broadcaster) persistLoop() {
	t := time.NewTicker(b.persistPeriod)
	defer t.Stop()
	for {
		select {
		case <-b.stop:
			return
		case <-t.C:
			b.mu.RLock()
			id := b.lastID
			b.mu.RUnlock()
			if id != "" && id != "0-0" {
				_ = b.client.Set(context.Background(), "balto:lastid:"+b.group, id, 0).Err()
			}
		}
	}
}

func (b *Broadcaster) Close(ctx context.Context) error {
	close(b.stop)
	done := make(chan struct{})
	go func() {
		b.stoppedWg.Wait()
		b.persistWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		b.mu.RLock()
		id := b.lastID
		b.mu.RUnlock()
		if id != "" && id != "0-0" {
			_ = b.client.Set(context.Background(), "balto:lastid:"+b.group, id, 0).Err()
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
