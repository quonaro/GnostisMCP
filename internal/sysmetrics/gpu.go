package sysmetrics

// collectGPUs tries each GPU backend in order: NVIDIA, AMD, Intel.
// Returns metrics from the first backend that succeeds, or nil if
// no GPU monitoring tool is available.
func collectGPUs() []GPUMetrics {
	if gpus := collectNvidiaGPUs(); gpus != nil {
		return gpus
	}
	if gpus := collectAMDGPUs(); gpus != nil {
		return gpus
	}
	if gpus := collectIntelGPUs(); gpus != nil {
		return gpus
	}
	return nil
}
