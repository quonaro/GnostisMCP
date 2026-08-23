package sysmetrics

import (
	"errors"
	"os/exec"
)

// firstAvailable returns the first path from paths that resolves to an
// executable via exec.LookPath. If none resolve, returns the last error.
func firstAvailable(paths []string) (string, error) {
	var lastErr error
	for _, p := range paths {
		resolved, err := exec.LookPath(p)
		if err == nil {
			return resolved, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no executable found")
	}
	return "", lastErr
}

// execFirstAvailable tries each candidate path in order, running the first
// one that exists and can be started. Returns the stdout output of the
// first successful command. If none succeed, returns the last error.
func execFirstAvailable(paths []string, args ...string) ([]byte, error) {
	var lastErr error
	for _, p := range paths {
		cmd := exec.Command(p, args...)
		output, err := cmd.Output()
		if err == nil {
			return output, nil
		}
		lastErr = err
	}
	return nil, lastErr
}
