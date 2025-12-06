package core

import (
	"time"

	"github.com/diabeney/balto/pkg/utils"
)

type ServiceInfo struct {
	ID         string   `json:"id"`
	Domain     string   `json:"domain"`
	PathPrefix string   `json:"path_prefix"`
	Ports      []string `json:"ports"`
}

type HealthStats struct {
	Status    string              `json:"status"`
	Uptime    string              `json:"uptime"`
	StartTime time.Time           `json:"start_time"`
	Services  int                 `json:"services"`
	Resources utils.ResourceUsage `json:"resources"`
}

type ServiceRequest struct {
	Domain     string   `json:"domain"`
	PathPrefix string   `json:"path_prefix"`
	Ports      []string `json:"ports"`
}
