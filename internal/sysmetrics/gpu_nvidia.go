package sysmetrics

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
)

// collectNvidiaGPUs queries nvidia-smi for NVIDIA GPU utilization.
// Returns nil if nvidia-smi is not available or no GPUs are found.
func collectNvidiaGPUs() []GPUMetrics {
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=index,name,utilization.gpu,memory.used,memory.total,temperature.gpu",
		"--format=csv,noheader,nounits",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil
	}
	if err := cmd.Start(); err != nil {
		return nil
	}

	var gpus []GPUMetrics
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 6 {
			continue
		}

		idx, _ := strconv.Atoi(strings.TrimSpace(fields[0]))
		name := strings.TrimSpace(fields[1])
		util, _ := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
		memUsed, _ := strconv.ParseUint(strings.TrimSpace(fields[3]), 10, 64)
		memTotal, _ := strconv.ParseUint(strings.TrimSpace(fields[4]), 10, 64)
		temp, _ := strconv.ParseFloat(strings.TrimSpace(fields[5]), 64)

		gpus = append(gpus, GPUMetrics{
			Index:              idx,
			Name:               name,
			UtilizationPercent: util,
			MemoryUsedBytes:    memUsed * 1024 * 1024, // MiB to bytes
			MemoryTotalBytes:   memTotal * 1024 * 1024,
			TemperatureC:       temp,
		})
	}

	if err := scanner.Err(); err != nil {
		_ = cmd.Wait()
		return nil
	}
	_ = cmd.Wait()
	return gpus
}
