package sysmetrics

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Metrics holds system resource usage.
type Metrics struct {
	CPU    CPUMetrics    `json:"cpu"`
	Memory MemoryMetrics `json:"memory"`
	GPUs   []GPUMetrics  `json:"gpus,omitempty"`
}

// CPUMetrics holds CPU usage information.
type CPUMetrics struct {
	Name         string  `json:"name"`
	UsagePercent float64 `json:"usage_percent"`
	Cores        int     `json:"cores"`
}

// MemoryMetrics holds RAM usage information.
type MemoryMetrics struct {
	Type         string  `json:"type,omitempty"`
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

// GPUMetrics holds GPU usage information for a single GPU.
type GPUMetrics struct {
	Index              int     `json:"index"`
	Name               string  `json:"name"`
	UtilizationPercent float64 `json:"utilization_percent"`
	MemoryUsedBytes    uint64  `json:"memory_used_bytes"`
	MemoryTotalBytes   uint64  `json:"memory_total_bytes"`
	TemperatureC       float64 `json:"temperature_c"`
}

// Collector collects system metrics, keeping state between calls
// to compute CPU usage deltas.
type Collector struct {
	mu       sync.Mutex
	lastIdle uint64
	lastSum  uint64
	lastTime time.Time
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	return &Collector{}
}

// Collect gathers current system metrics. The first call returns 0% CPU
// usage because a delta between two reads is required.
func (c *Collector) Collect() Metrics {
	return Metrics{
		CPU:    c.collectCPU(),
		Memory: collectMemory(),
		GPUs:   collectGPUs(),
	}
}

func (c *Collector) collectCPU() CPUMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	idle, sum, ok := readCPUStats()
	if !ok {
		return CPUMetrics{Cores: runtime.NumCPU()}
	}

	usage := 0.0
	if !c.lastTime.IsZero() {
		dt := now.Sub(c.lastTime).Seconds()
		if dt > 0 {
			dIdle := float64(idle - c.lastIdle)
			dSum := float64(sum - c.lastSum)
			if dSum > 0 {
				usage = (1 - dIdle/dSum) * 100
			}
		}
	}

	c.lastIdle = idle
	c.lastSum = sum
	c.lastTime = now

	return CPUMetrics{
		Name:         cpuModelName(),
		UsagePercent: usage,
		Cores:        runtime.NumCPU(),
	}
}

// cpuModelName reads the CPU model name from /proc/cpuinfo.
func cpuModelName() string {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	_ = scanner.Err()
	return ""
}

// ramType detects the RAM type (DDR3, DDR4, DDR5, etc.) via dmidecode.
// Returns "" if dmidecode is not available or not permitted.
func ramType() string {
	cmd := exec.Command("dmidecode", "-t", "memory")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Type:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Type:"))
			if val != "" && val != "Unknown" && val != "Unknown " {
				return val
			}
		}
	}
	_ = scanner.Err()
	return ""
}

// readCPUStats reads /proc/stat and returns (idle, total, ok).
func readCPUStats() (idle, total uint64, ok bool) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return 0, 0, false
	}
	line := scanner.Text()
	if !strings.HasPrefix(line, "cpu ") {
		return 0, 0, false
	}

	fields := strings.Fields(line[4:])
	if len(fields) < 4 {
		return 0, 0, false
	}

	var sum uint64
	for _, f := range fields {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		sum += v
	}

	// idle is field[3], iowait is field[4] (if present)
	idleVal, _ := strconv.ParseUint(fields[3], 10, 64)
	return idleVal, sum, true
}

func collectMemory() MemoryMetrics {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return MemoryMetrics{}
	}
	defer func() { _ = f.Close() }()

	var total, available uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(parts[1], 10, 64)
		switch parts[0] {
		case "MemTotal:":
			total = val * 1024 // kB to bytes
		case "MemAvailable:":
			available = val * 1024
		}
	}

	if err := scanner.Err(); err != nil {
		return MemoryMetrics{}
	}

	used := total - available
	pct := 0.0
	if total > 0 {
		pct = float64(used) / float64(total) * 100
	}

	return MemoryMetrics{
		Type:         ramType(),
		TotalBytes:   total,
		UsedBytes:    used,
		UsagePercent: pct,
	}
}
