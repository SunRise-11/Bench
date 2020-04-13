package system

import (
	"fmt"
	"time"

	disk "github.com/mackerelio/go-osstat/memory"
	gostats "github.com/shirou/gopsutil/cpu"
	gomem "github.com/shirou/gopsutil/mem"
)

// CPU interface for CPU related details.
type CPU interface {
	GetTotalCPUUsage() string
	GetTotalCPUStats() gostats.TimesStat
}

// Memory interface for Memory related details.
type Memory interface {
	GetVirtualMemoryStats() MemoryStats
}

// Disk interface for Disk related details.
type Disk interface {
	GetDiskIOStats() DiskStats
}

// SystemMetrics contains system level metrics.
// These are very basic as of now and can be
// made to collect advanced informations with
// the same implementation.
type SystemMetrics struct{}

// New returns a new system-metrics reader.
func New() *SystemMetrics {
	return &SystemMetrics{}
}

// GetTotalCPUUsage returns the total CPU usage by all the
// available cores over the previous second.
func (s *SystemMetrics) GetTotalCPUUsage(c chan string) {
	usage, err := gostats.Percent(time.Duration(time.Second), false)
	if err != nil {
		panic(err)
	}

	c <- fmt.Sprintf("%.2f", usage[0])
}

// GetTotalCPUStats returns the stats related to the CPU
// as a whole.
func (s *SystemMetrics) GetTotalCPUStats() gostats.TimesStat {
	stats, err := gostats.Times(false)
	if err != nil {
		panic(err)
	}

	return stats[0]
}

// MemoryStats for memory based statistics.
type MemoryStats struct {
	Total       uint64  `json:"totalBytes"`
	Available   uint64  `json:"availableBytes"`
	Used        uint64  `json:"usedBytes"`
	UsedPercent float64 `json:"usedPercent"`
	Free        uint64  `json:"freeBytes"`
}

// GetVirtualMemoryStats returns the memory statistics of the host machine.
func (s *SystemMetrics) GetVirtualMemoryStats(c chan MemoryStats) {
	stats, err := gomem.VirtualMemory()
	if err != nil {
		panic(err)
	}

	c <- MemoryStats{
		// default is always in bytes. hence, convert into the required format.
		Total:       stats.Total / 1000000,
		Available:   stats.Available / 1000000,
		Used:        stats.Used / 1000000,
		UsedPercent: stats.UsedPercent,
		Free:        stats.Free / 1000000,
	}
}

// DiskStats statistics for information related to the disk.
type DiskStats struct {
	DiskIO int `json:"diskIO"`
	Cached int `json:"cached"`
}

// GetDiskIOStats returns the disk stats: IO per sec and cached volume.
func (s *SystemMetrics) GetDiskIOStats(c chan DiskStats) {
	before, err := disk.Get()
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Duration(time.Second * 3))

	now, err := disk.Get()
	if err != nil {
		panic(err)
	}

	c <- DiskStats{
		// default is always in bytes. hence, convert into the required format.
		DiskIO: (int(now.Used) - int(before.Used)) / 1000, // in kilo-bytes
		Cached: int(now.Cached / 1000000),                 // mega-bytes
	}
}

// Encode encodes the blocks into format that can be consumed
// by the tsdb module.
func (s *SystemMetrics) Encode(block interface{}) string {
	switch node := block.(type) {
	case DiskStats:
		return fmt.Sprintf("%d|%d", node.DiskIO, node.Cached)
	case MemoryStats:
		return fmt.Sprintf("%d|%d|%d|%f|%d",
			node.Total, node.Available, node.Used, node.UsedPercent, node.Free,
		)
	case string:
		return node
	}

	data, ok := block.(string)
	if !ok {
		panic(fmt.Sprintf("Invalid block type: %v", block))
	}

	return data
}
