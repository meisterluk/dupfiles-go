package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dupfiles",
	Short: "Determine duplicate files and folders",
	Long: `dupfiles-go is a program with a command-line interface. It allows users to
• generate reports of a filesystem state
• find the highest duplicate nodes in two or more reports

Thus, this implementation allows you to find duplicate nodes on your filesystem.
This implementation is written in Go and implements the ‘dupfiles 1.0’ specification.
`,
}

func init() {
	cobra.OnInitialize(initConfig)

	// TODO rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.test-cobra.yaml)")
	rootCmd.PersistentFlags().BoolVar(&argConfigOutput, "config", false, "only prints the configuration and terminates")
	rootCmd.PersistentFlags().BoolVar(&argJSONOutput, "json", false, "return output as JSON, not as plain text")

	// Cobra also supports local flags, which will only run
	// when this action is called directly. TODO remove
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	/*if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".test-cobra" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".test-cobra")
	}*/

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// Execute executes the command line given in $args.
// It writes the result to Output $w and errors/information messages to $log.
// It returns a triple (exit code, JSON output?, error)
func Execute(args []string, wLocal Output, logLocal Output) (int, bool, error) {
	// NOTE we (sadly) use global variables, because cobra does not provide
	//      mechanisms to pass values acc. to my style of writing CLI applications

	// NOTE global input variables: {w, log}
	w = wLocal
	log = logLocal

	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	if err == nil && cmdError != nil {
		err = cmdError
	}

	// NOTE global output variables: {exitCode, argJSONOutput}
	return exitCode, argJSONOutput, err
}

func main() {
	// this output stream will be filled with text or JSON (if --json) output
	output := PlainOutput{Device: os.Stdout}
	// this output stream will be used for status messages
	logOutput := PlainOutput{Device: os.Stderr}

	exitcode, jsonOutput, err := Execute(os.Args[1:], &output, &logOutput)
	// TODO verify that all input files are properly UTF-8 encoded ⇒ output if properly UTF-8 encoded

	if err == nil {
		os.Exit(exitcode)
	}

	if jsonOutput {
		type jsonError struct {
			Message  string `json:"error"`
			ExitCode int    `json:"code"`
		}

		jsonData := jsonError{
			Message:  err.Error(),
			ExitCode: exitcode,
		}
		jsonRepr, err := json.Marshal(&jsonData)
		if err != nil {
			// ignore return value intentionally, as we cannot do anything about it
			logOutput.Printfln(`error (exitcode=%d) was thrown: %s`, exitcode, err.Error())
			logOutput.Printfln(`but I failed to create a response in JSON`)
			output.Println(`{"error":"could not encode error message as JSON","exitcode":6}`)
			exitcode = 6
		} else {
			output.Println(string(jsonRepr))
		}
	} else {
		output.Printfln("Error: %s", err.Error())
	}

	os.Exit(exitcode)
}
