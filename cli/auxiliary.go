package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type jsonError struct {
	Message  string `json:"error"`
	ExitCode int    `json:"code"`
}

// handleError prints the error in the appropriate format
func handleError(msg string, exitCode int, jsonOutput bool) {
	if jsonOutput {
		jErr := jsonError{msg, exitCode}
		jsonRepr, err := json.Marshal(jErr)
		if err != nil {
			fmt.Fprintln(os.Stderr, `{"error":"could not encode error message as JSON","exitcode":2}`)
			os.Exit(2)
		} else {
			fmt.Fprintln(os.Stderr, jsonRepr)
			os.Exit(exitCode)
		}
	}
	fmt.Fprintln(os.Stderr, `Error: `+msg)
	os.Exit(exitCode)
}

// envOr returns either environment variable envKey (if non-empty) or the default Value
func envOr(envKey, defaultValue string) string {
	val, ok := os.LookupEnv(envKey)
	if !ok || val == "" {
		return defaultValue
	}
	return envKey
}

// envToBool returns environment variable envKey considered as boolean value
func envToBool(envKey string) bool {
	val, ok := os.LookupEnv(envKey)
	return (ok && val == "1" || ok && strings.ToLower(val) == "true")
}

// envToInt returns environment variable envKey considered as integer value
func envToInt(envKey string) (int, bool) {
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

// Was the JSON output format requested?
func jsonOutput() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--json" {
			return true
		}
	}
	return false
}
