package sysmetrics

import (
	"bufio"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// collectIntelGPUs queries xpu-smi for Intel GPU utilization.
// Returns nil if xpu-smi is not available or no GPUs are found.
func collectIntelGPUs() []GPUMetrics {
	// List devices to get count and names.
	cmd := exec.Command("xpu-smi", "discovery", "-l")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var gpus []GPUMetrics
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		// Lines look like: "0 | Intel(R) Arc(TM) A770 Graphics | ..."
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}
		idx, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(parts[1])
		if name == "" {
			name = "Intel GPU"
		}

		util, memUsed, memTotal, temp := collectIntelGPUStats(idx)
		gpus = append(gpus, GPUMetrics{
			Index:              idx,
			Name:               name,
			UtilizationPercent: util,
			MemoryUsedBytes:    memUsed,
			MemoryTotalBytes:   memTotal,
			TemperatureC:       temp,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil
	}
	return gpus
}

// collectIntelGPUStats queries per-device stats from xpu-smi stats.
func collectIntelGPUStats(idx int) (util float64, memUsed, memTotal uint64, temp float64) {
	cmd := exec.Command("xpu-smi", "stats", "-d", strconv.Itoa(idx))
	output, err := cmd.Output()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Lines look like: "EU Array Active (%) | 50.0"
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch {
		case strings.Contains(strings.ToLower(key), "eu array active"):
			util = parseFloatSafe(val)
		case strings.Contains(strings.ToLower(key), "memory used"):
			memUsed = parseUintFromStr(val) * 1024 * 1024 // MiB to bytes
		case strings.Contains(strings.ToLower(key), "memory total"):
			memTotal = parseUintFromStr(val) * 1024 * 1024
		case strings.Contains(strings.ToLower(key), "temperature"):
			temp = parseFloatSafe(val)
		}
	}

	return
}

var numRe = regexp.MustCompile(`[0-9]+\.?[0-9]*`)

func parseFloatSafe(s string) float64 {
	match := numRe.FindString(s)
	if match == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(match, 64)
	return f
}

func parseUintFromStr(s string) uint64 {
	match := numRe.FindString(s)
	if match == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(match, 64)
	return uint64(f)
}
