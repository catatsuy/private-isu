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

var scoreTotal = NewScore()

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
	quitC := make(chan bool)
	quit := false

	workersC := make(chan *Worker, 20)

	go func() {
		for {
			workersC <- NewWorker(target)

			if quit {
				break
			}
		}
	}()

	go func() {
		// for stopping goroutines
		<-quitC
		quit = true
	}()

	toppageNotLogin := NewScenario("GET", "/me")
	toppageNotLogin.ExpectedStatusCode = 200
	toppageNotLogin.ExpectedLocation = "/"

	go func() {
		// not login
		for {
			toppageNotLogin.Play(<-workersC)

			if quit {
				break
			}
		}
	}()

	login := NewScenario("POST", "/login")
	login.ExpectedStatusCode = 200
	login.ExpectedLocation = "/"

	mepage := NewScenario("GET", "/me")
	mepage.ExpectedStatusCode = 200
	mepage.ExpectedLocation = "/me"

	go func() {
		for {
			login.PostData = map[string]string{
				"account_name": "catatsuy",
				"password":     "kaneko",
			}
			w := <-workersC
			login.Play(w)
			mepage.Play(w)

			if quit {
				break
			}
		}
	}()

	<-ec
	quitC <- true

	var errs []error

	fmt.Printf("score: %d, suceess: %d, fail: %d\n",
		scoreTotal.GetScore(),
		scoreTotal.GetSucesses(),
		scoreTotal.GetFails(),
	)

	for _, err := range errs {
		fmt.Println(err)
	}

	return ExitCodeOK
}
