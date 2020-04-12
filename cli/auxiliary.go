package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// EnvOr returns either environment variable envKey (if non-empty) or the default Value
func EnvOr(envKey, defaultValue string) string {
	val, ok := os.LookupEnv(envKey)
	if !ok || val == "" {
		return defaultValue
	}
	return envKey
}

// EnvToBool returns environment variable envKey considered as boolean value
func EnvToBool(envKey string) (bool, error) {
	val, ok := os.LookupEnv(envKey)
	if ok && (val == `1` || strings.ToLower(val) == `true`) {
		return true, nil
	} else if ok && (val == `0` || strings.ToLower(val) == `false`) {
		return false, nil
	}
	return false, fmt.Errorf(`boolean env key '%s' has non-bool value '%s'`, envKey, val)
}

// EnvToInt returns environment variable envKey considered as integer value
func EnvToInt(envKey string) (int, bool) {
	val, ok := os.LookupEnv(envKey)
	if !ok {
		return 0, false
	}
	i, err := strconv.ParseUint(val, 10, 16)
	if err != nil || i <= 0 || i > 256 {
		return 0, false
	}
	return int(i), true
}

// CountCPUs determines the number of logical CPUs in this machine
func CountCPUs() int {
	return runtime.NumCPU()
}
