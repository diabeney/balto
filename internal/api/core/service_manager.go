package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/diabeney/balto/internal/router"
	"github.com/diabeney/balto/pkg/utils"
)

type ServiceManager struct {
	mu        sync.RWMutex
	services  map[string]*ServiceInfo
	startTime time.Time
}

var (
	serviceManager *ServiceManager
	once           sync.Once
)

func InitServiceManager() {
	once.Do(func() {
		serviceManager = &ServiceManager{
			services:  make(map[string]*ServiceInfo),
			startTime: time.Now(),
		}
	})
}

func GetServiceManager() *ServiceManager {
	return serviceManager
}

func (sm *ServiceManager) InitializeFromConfig(services []ServiceInfo) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, svc := range services {
		s := svc
		sm.services[svc.ID] = &s
	}

	return nil
}

func (sm *ServiceManager) ListServices() []*ServiceInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	services := make([]*ServiceInfo, 0, len(sm.services))
	for _, svc := range sm.services {
		services = append(services, svc)
	}
	return services
}

func (sm *ServiceManager) GetService(id string) (*ServiceInfo, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	svc, ok := sm.services[id]
	return svc, ok
}

func (sm *ServiceManager) AddService(domain, pathPrefix string, ports []string) (*ServiceInfo, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := utils.HashServiceID(domain, pathPrefix)
	if _, exists := sm.services[id]; exists {
		return nil, fmt.Errorf("service already exists: %s", id)
	}

	svc := &ServiceInfo{
		ID:         id,
		Domain:     domain,
		PathPrefix: pathPrefix,
		Ports:      ports,
	}
	sm.services[id] = svc

	if err := sm.rebuildRouter(); err != nil {
		delete(sm.services, id)
		return nil, err
	}

	return svc, nil
}

func (sm *ServiceManager) rebuildRouter() error {
	services := make([]router.ServiceInfo, 0, len(sm.services))
	for _, svc := range sm.services {
		services = append(services, router.ServiceInfo{
			ID:         svc.ID,
			Domain:     svc.Domain,
			PathPrefix: svc.PathPrefix,
			Ports:      svc.Ports,
		})
	}
	return router.RebuildFromServices(services)
}

func (sm *ServiceManager) UpdateService(id string, req *ServiceRequest) (*ServiceInfo, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	oldSvc, exists := sm.services[id]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", id)
	}

	newID := utils.HashServiceID(req.Domain, req.PathPrefix)
	if newID != id {
		if _, exists := sm.services[newID]; exists {
			return nil, fmt.Errorf("service with new ID already exists: %s", newID)
		}
		delete(sm.services, id)
	}

	svc := &ServiceInfo{
		ID:         newID,
		Domain:     req.Domain,
		PathPrefix: req.PathPrefix,
		Ports:      req.Ports,
	}
	sm.services[newID] = svc

	if err := sm.rebuildRouter(); err != nil {
		if newID != id {
			delete(sm.services, newID)
			sm.services[id] = oldSvc
		} else {
			sm.services[id] = oldSvc
		}
		return nil, err
	}

	return svc, nil
}

func (sm *ServiceManager) GetHealthStats() *HealthStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	uptime := time.Since(sm.startTime)
	return &HealthStats{
		Status:    "ok",
		Uptime:    uptime.String(),
		StartTime: sm.startTime,
		Services:  len(sm.services),
		Resources: utils.GetResourceUsage(),
	}
}
