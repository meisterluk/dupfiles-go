package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/meisterluk/dupfiles-go/internals"
)

type TestString func(exp *Expect, text string) (string, error)

var testStringFunctions map[string]TestString = map[string]TestString{
	`isJSON`: func(exp *Expect, text string) (string, error) {
		var void map[string]interface{}
		if json.Unmarshal([]byte(text), &void) == nil {
			return `is JSON content`, nil
		}
		return `is JSON content`, fmt.Errorf(`expected JSON content, got '%s'`, text)
	},
	`isUTF8`: func(exp *Expect, text string) (string, error) {
		if utf8.Valid([]byte(text)) {
			return `is valid UTF8 content`, nil
		}
		return `is valid UTF8 content`, fmt.Errorf(`expected valid UTF8 content, got '%s'`, internals.ByteEncode(text))
	},
}

type Expect struct {
	Env            map[string]string
	StdinSend      string
	StdoutContains string
	StdoutIs       string
	StdoutTest     []TestString
	StderrContains string
	StderrIs       string
	StderrTest     []TestString
	MaxDuration    float64
	ExitCode       int
}

func success(msg string, args ...interface{}) string {
	return fmt.Sprintf(fmt.Sprintf("✓ %s", msg), args...)
}

func fail(msg string, args ...interface{}) string {
	return fmt.Sprintf(fmt.Sprintf("✗ %s", msg), args...)
}

func short(msg string) string {
	msg = strings.ReplaceAll(msg, "\n", `\n`)
	if len(msg) > 10 {
		return msg[0:10] + " …"
	}
	return msg
}

func NewExpect() *Expect {
	e := new(Expect)
	e.Env = make(map[string]string)
	e.MaxDuration = -1.0
	e.ExitCode = -1
	return e
}

func run(name string, t *testing.T, executable string, check *Expect, args ...string) {
	// create Command
	cmd := exec.Command(executable, args...)

	// set stdin content
	if check.StdinSend != "" {
		cmd.Stdin = strings.NewReader(check.StdinSend)
	}

	// set env
	if len(check.Env) != 0 {
		cmd.Env = os.Environ()
		for key, value := range check.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	t.Logf("== starting %s ==", name)

	// actually Run the command
	before := time.Now()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitCode int
	if err != nil {
		if strings.HasPrefix(err.Error(), "exit status") {
			e, err2 := strconv.Atoi(err.Error()[12:])
			if err2 != nil {
				t.Fatalf("exit code parsing failed; '%q'; %s", err.Error(), err2)
			}
			exitCode = e
		} else {
			t.Fatalf(err.Error())
		}
	}
	duration := time.Since(before)

	// check stdout
	if check.StdoutContains != "" {
		if strings.Contains(stdout.String(), check.StdoutContains) {
			t.Log(success("stdout contains '%s'", short(check.StdoutContains)))
		} else {
			t.Error(fail("stdout misses '%s'", short(check.StdoutContains)))
		}
	}
	if check.StdoutIs != "" {
		if stdout.String() == check.StdoutIs {
			t.Log(success("stdout is '%s'", short(check.StdoutIs)))
		} else {
			t.Error(fail("stdout is not '%s'", short(check.StdoutIs)))
		}
	}
	if check.StdoutTest != nil {
		for _, testFunc := range check.StdoutTest {
			desc, err := testFunc(check, stdout.String())
			if err != nil {
				t.Error(fail("stdout failed functional test '%s'", desc))
			} else {
				t.Log(success("stdout passed functional test '%s'", desc))
			}
		}
	}

	// check stderr
	if check.StderrContains != "" {
		if strings.Contains(stderr.String(), check.StderrContains) {
			t.Log(success("stderr contains '%s'", short(check.StderrContains)))
		} else {
			t.Error(fail("stderr misses '%s'", short(check.StderrContains)))
		}
	}
	if check.StderrIs != "" {
		if stderr.String() == check.StderrIs {
			t.Log(success("stderr is '%s'", short(check.StderrIs)))
		} else {
			t.Error(fail("stderr is not '%s'", short(check.StderrIs)))
		}
	}
	if check.StderrTest != nil {
		for _, testFunc := range check.StderrTest {
			desc, err := testFunc(check, stderr.String())
			if err != nil {
				t.Error(fail("stderr failed functional test '%s'", desc))
			} else {
				t.Log(success("stderr passed functional test '%s'", desc))
			}
		}
	}

	// check duration
	if check.MaxDuration >= 0.0 {
		if float64(duration)/float64(time.Second) > check.MaxDuration {
			t.Error(fail("execution took %v, but max is %f", duration, check.MaxDuration))
		} else {
			t.Log(success("execution time %v is below %.2f", duration, check.MaxDuration))
		}
	}

	// check exit code
	if check.ExitCode >= 0 {
		if exitCode == check.ExitCode {
			t.Log(success("exitcode is indeed %d", check.ExitCode))
		} else {
			t.Error(fail("expected exitcode %d, got %d", check.ExitCode, exitCode))
		}
	}
}
