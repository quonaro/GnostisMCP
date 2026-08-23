package sysmetrics

import (
	"encoding/json"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// collectAMDGPUs queries rocm-smi for AMD GPU utilization.
// Returns nil if rocm-smi is not available or no GPUs are found.
func collectAMDGPUs() []GPUMetrics {
	cmd := exec.Command("rocm-smi",
		"--showuse", "--showmeminfo", "vram", "--showtemp", "--showproductname", "--json",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var raw map[string]map[string]string
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil
	}

	var gpus []GPUMetrics
	for key, props := range raw {
		if !strings.HasPrefix(key, "card") {
			continue
		}
		idxStr := strings.TrimPrefix(key, "card")
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			continue
		}

		name := firstNonEmpty(props["Card Series"], props["GPU name"], props["Card series"], props["Card model"], "AMD GPU")
		util := parseFloatProp(props, "GPU use (%)")
		memUsed := parseUintProp(props, "VRAM Total Used Memory (B)", "GPU memory used (VRAM%)", "GPU memory used (MB)", "VRAM used (bytes)")
		memTotal := parseUintProp(props, "VRAM Total Memory (B)", "GPU memory total (VRAM%)", "GPU memory total (MB)", "VRAM total (bytes)")
		temp := parseFloatProp(props, "Temperature (Sensor edge) (C)", "Edge Temp (C)")

		// rocm-smi reports memory in bytes; convert if values look like MB
		memUsedBytes, memTotalBytes := normalizeMem(memUsed, memTotal)

		gpus = append(gpus, GPUMetrics{
			Index:              idx,
			Name:               name,
			UtilizationPercent: util,
			MemoryUsedBytes:    memUsedBytes,
			MemoryTotalBytes:   memTotalBytes,
			TemperatureC:       temp,
		})
	}

	sort.Slice(gpus, func(i, j int) bool { return gpus[i].Index < gpus[j].Index })
	return gpus
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

func parseFloatProp(props map[string]string, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := props[k]; ok {
			if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				return f
			}
		}
	}
	return 0
}

func parseUintProp(props map[string]string, keys ...string) uint64 {
	for _, k := range keys {
		if v, ok := props[k]; ok {
			if n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64); err == nil {
				return n
			}
		}
	}
	return 0
}

// normalizeMem adjusts memory values: if values are small enough to be MB,
// convert to bytes. rocm-smi typically reports VRAM in bytes.
func normalizeMem(used, total uint64) (uint64, uint64) {
	const mbThreshold = 1024 * 1024 // if < 1MB worth of bytes, assume MB
	if total > 0 && total < mbThreshold {
		return used * 1024 * 1024, total * 1024 * 1024
	}
	return used, total
}
