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

	go checkLoop(workers)

	<-ec

	var totalScore int64
	var totalSuccesses int32
	var totalFails int32
	var errs []error

	for _, w := range workers {
		totalScore += w.Score
		totalSuccesses += w.Successes
		totalFails += w.Fails
		errs = append(errs, w.Errors...)
	}

	fmt.Printf("score: %d, suceess: %d, fail: %d\n", totalScore, totalSuccesses, totalFails)

	for _, err := range errs {
		fmt.Println(err)
	}

	return ExitCodeOK
}

func checkLoop(workers []*Worker) {
	toppageNotLogin := NewScenario("GET", "/me")
	toppageNotLogin.ExpectedStatusCode = 200
	toppageNotLogin.ExpectedLocation = "/"

	login := NewScenario("POST", "/login")

	toppage := NewScenario("GET", "/me")
	toppage.ExpectedStatusCode = 200

	for {
		toppageNotLogin.Play(workers[0])

		login.PostData = map[string]string{
			"account_name": "catatsuy",
			"password":     "kaneko",
		}
		login.Play(workers[1])
		toppage.Play(workers[1])
	}
}
