package routines

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// ---------- Helpers for parsing k8s resource strings ----------
var (
	cpuRegex    = regexp.MustCompile(`^([0-9]*\.?[0-9]+)(m|)$`)
	memoryRegex = regexp.MustCompile(`^([0-9]*\.?[0-9]+)(Ei|Pi|Ti|Gi|Mi|Ki|E|P|T|G|M|K|i|)$`)
)

// parseK8sCPU converts CPU strings like "100m", "250m", "1", "0.5" into nano cores (1 core = 1e9 nano cores)
// returns uint64 nano cores
func parseK8sCPU(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty cpu string")
	}
	// handle numeric with m
	if matches := cpuRegex.FindStringSubmatch(s); matches != nil {
		valueStr := matches[1]
		unit := matches[2] // "m" or ""
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return 0, err
		}
		if unit == "m" {
			// millicores: value m -> value/1000 cores
			cores := value / 1000.0
			nano := uint64(math.Round(cores * 1e9))
			return nano, nil
		}
		// plain cores (possibly decimal)
		nano := uint64(math.Round(value * 1e9))
		return nano, nil
	}
	// fallback: try plain float
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return uint64(math.Round(v * 1e9)), nil
	}
	return 0, fmt.Errorf("unrecognized cpu format: %q", s)
}

// parseK8sMemory converts Kubernetes memory strings like "128Mi", "1Gi", "512Ki", "1000000" into bytes
func parseK8sMemory(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty memory string")
	}
	// Accept numeric bytes
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return uint64(math.Round(v)), nil
	}
	// Try units with Ki/Mi/Gi etc.
	// Accept uppercase suffixes like Mi, Gi, K, M, G as in many k8s outputs
	re := regexp.MustCompile(`(?i)^([0-9]*\.?[0-9]+)\s*(k|m|g|t|p|e|ki|mi|gi|ti|pi|ei)?b?$`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("unrecognized memory format: %q", s)
	}
	value, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, err
	}
	unit := strings.ToLower(m[2])
	switch unit {
	case "ki":
		return uint64(value * 1024.0), nil
	case "mi":
		return uint64(value * 1024.0 * 1024.0), nil
	case "gi":
		return uint64(value * 1024.0 * 1024.0 * 1024.0), nil
	case "ti":
		return uint64(value * 1024.0 * 1024.0 * 1024.0 * 1024.0), nil
	case "pi":
		return uint64(value * math.Pow(1024.0, 5)), nil
	case "ei":
		return uint64(value * math.Pow(1024.0, 6)), nil
	case "k":
		return uint64(value * 1000.0), nil
	case "m":
		return uint64(value * 1000.0 * 1000.0), nil
	case "g":
		return uint64(value * 1000.0 * 1000.0 * 1000.0), nil
	case "t":
		return uint64(value * math.Pow(1000.0, 4)), nil
	case "p":
		return uint64(value * math.Pow(1000.0, 5)), nil
	case "e":
		return uint64(value * math.Pow(1000.0, 6)), nil
	case "":
		// no unit matched but regex matched earlier numeric
		return uint64(value), nil
	default:
		return 0, fmt.Errorf("unrecognized memory unit: %q", unit)
	}
}
