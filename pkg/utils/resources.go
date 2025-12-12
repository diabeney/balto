package utils

import (
	"runtime"
)

type ResourceUsage struct {
	Memory MemoryStats `json:"memory"`
	CPU    CPUStats    `json:"cpu"`
}

type MemoryStats struct {
	Alloc      uint64 `json:"alloc"`       // bytes allocated and not yet freed
	TotalAlloc uint64 `json:"total_alloc"` // total bytes allocated (even if freed)
	Sys        uint64 `json:"sys"`         // bytes obtained from system
	NumGC      uint32 `json:"num_gc"`      // number of garbage collections
	HeapAlloc  uint64 `json:"heap_alloc"`  // bytes allocated and not yet freed (same as Alloc above)
	HeapSys    uint64 `json:"heap_sys"`    // bytes obtained from system for heap
}

type CPUStats struct {
	NumCPU       int `json:"num_cpu"`       // number of logical CPUs
	NumGoroutine int `json:"num_goroutine"` // number of goroutines
}

func GetResourceUsage() ResourceUsage {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return ResourceUsage{
		Memory: MemoryStats{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
			HeapAlloc:  m.HeapAlloc,
			HeapSys:    m.HeapSys,
		},
		CPU: CPUStats{
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
		},
	}
}
