package main

import (
	"flag"
	"fmt"
	"io"
	"time"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota
)

// CLI is the command line object
type CLI struct {
	// outStream and errStream are the stdout and stderr
	// to write message from the CLI.
	outStream, errStream io.Writer
}

// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	var (
		target string

		version bool
	)

	// Define option flag parse
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)

	flags.StringVar(&target, "target", "", "")
	flags.StringVar(&target, "t", "", "(Short)")

	flags.BoolVar(&version, "version", false, "Print version information and quit.")

	// Parse commandline flag
	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeError
	}

	// Show version
	if version {
		fmt.Fprintf(cli.errStream, "%s version %s\n", Name, Version)
		return ExitCodeOK
	}

	ec := time.After(10 * time.Second)

	go func() {
		<-ec
	}()

	workers := []*Worker(make([]*Worker, 5))

	for i := 0; i < 5; i++ {
		workers[i] = NewWorker(target)
	}

	go loadLoop(workers)

	<-ec

	var totalScore int64
	var totalSuccesses int32

	for _, w := range workers {
		totalScore += w.Score
		totalSuccesses += w.Successes
	}

	fmt.Printf("score: %d, suceess: %d\n", totalScore, totalSuccesses)

	return ExitCodeOK
}

func checkLoop() {

}

func loadLoop(workers []*Worker) {
	toppage := NewScenario("GET", "/me")
	login := NewScenario("POST", "/login")

	for {
		for _, worker := range workers {
			toppage.Play(worker)

			login.PostData = map[string]string{
				"account_name": "catatsuy",
				"password":     "kaneko",
			}
			login.Play(worker)
			toppage.Play(worker)
		}
	}
}
