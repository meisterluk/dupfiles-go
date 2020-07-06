package tests

import (
	"fmt"
	"os"
	"testing"
)

func TestLs(t *testing.T) {
	// get path to executable
	executable := os.Getenv("EXEC")
	if executable == "" {
		t.Fatalf("executable not found; please set env EXEC to point to the dupfiles executable")
	}

	// example call
	exp := NewExpect()
	exp.StdoutContains = "."
	exp.ExitCode = 0
	run("ls has .", t, executable, exp, ".")

	// YAML spec support
	data := new(YAMLTestsSpec)
	err := parseYAMLTestSpec([]byte(testYAML), data)
	if err != nil {
		fmt.Println(err)
	}

	data.Executable = executable // is overwritten
	for _, spec := range data.Expected {
		exp = NewExpect()
		for key, val := range spec.Env {
			exp.Env[key] = val
		}
		exp.ExitCode = spec.ExitCode
		if spec.Runtime.Max != 0.0 {
			exp.MaxDuration = spec.Runtime.Max
		}
		if spec.Stdin.Is != "" {
			exp.StdinSend = spec.Stdin.Is
		}
		if spec.Stderr.Contains != "" {
			exp.StderrContains = spec.Stderr.Contains
		}
		if spec.Stderr.Is != "" {
			exp.StderrIs = spec.Stderr.Is
		}
		exp.StderrTest = make([]TestString, 0, 8)
		for _, method := range spec.Stderr.Apply {
			exp.StderrTest = append(exp.StderrTest, testStringFunctions[method])
		}
		if spec.Stdout.Contains != "" {
			exp.StdoutContains = spec.Stdout.Contains
		}
		if spec.Stdout.Is != "" {
			exp.StdoutIs = spec.Stdout.Is
		}
		exp.StdoutTest = make([]TestString, 0, 8)
		for _, method := range spec.Stdout.Apply {
			exp.StdoutTest = append(exp.StdoutTest, testStringFunctions[method])
		}
		run(spec.Name, t, executable, exp, spec.Args...)
	}

	fmt.Println(data)
}
