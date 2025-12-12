package metrics

import (
	"time"
)

type RequestTiming struct {
	DNSLookup        time.Duration `json:"-"`
	TCPConnection    time.Duration `json:"-"`
	TLSHandshake     time.Duration `json:"-"`
	ServerProcessing time.Duration `json:"-"`
	ContentTransfer  time.Duration `json:"-"`
	TTFB             time.Duration `json:"-"`
	Total            time.Duration `json:"-"`
}

type RequestMetric struct {
	Timestamp     time.Time     `json:"timestamp"`
	Method        string        `json:"method"`
	Route         string        `json:"route"`
	BackendID     string        `json:"backend_id"`
	StatusCode    int           `json:"status_code"`
	Timing        RequestTiming `json:"timing"`
	BytesSent     int64         `json:"bytes_sent"`
	BytesReceived int64         `json:"bytes_received"`
}
