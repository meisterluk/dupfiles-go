package tests

import (
	"gopkg.in/yaml.v2"
)

const exampleYAML = `
executable: echo

expected:
  - args: ['example', '--help']
    env:
      KEY1: VALUE1
      KEY2: VALUE2
    stdin:
      is: |
        some string
        sent to stdin once the executable has started
    stdout:
      is: |
        some expected exact match for stdout,
        not just some substring
      contains: |
        substring
    stderr:
      contains: |
        1.0
      apply:
        - isJSON
    runtime:
      max: 5.5
    exitcode: 0
`

const testYAML = `
expected:
  - name: 'some version test'
    args: ['version', '--help']
    env:
      KEY1: VALUE1
      KEY2: VALUE2
    stdin:
      is: |
        some string
        sent to stdin once the executable has started
    stdout:
      is: |
        some expected exact match for stdout,
        not just some substring
      contains: |
        substring
    stderr:
      contains: |
        1.0
      apply:
        - isJSON
    runtime:
      max: 5.5
    exitcode: 0
`

type YAMLInputStreamSpec struct {
	Is string `yaml:"is"`
}

type YAMLOutputStreamSpec struct {
	Is       string   `yaml:"is"`
	Contains string   `yaml:"contains"`
	Apply    []string `yaml:"apply"`
}

type YAMLRuntimeSpec struct {
	Max float64
}

type YAMLTestSpec struct {
	Name     string               `yaml:"name"`
	Args     []string             `yaml:"args"`
	Env      map[string]string    `yaml:"env"`
	Stdin    YAMLInputStreamSpec  `yaml:"stdin"`
	Stdout   YAMLOutputStreamSpec `yaml:"stdout"`
	Stderr   YAMLOutputStreamSpec `yaml:"stderr"`
	Runtime  YAMLRuntimeSpec      `yaml:"runtime"`
	ExitCode int                  `yaml:"exitcode"`
}

type YAMLTestsSpec struct {
	Executable string         `yaml:"executable"`
	Expected   []YAMLTestSpec `yaml:"expected"`
}

func parseYAMLTestSpec(src []byte, dst *YAMLTestsSpec) error {
	return yaml.Unmarshal(src, dst)
}
