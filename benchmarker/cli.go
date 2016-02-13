package main

import (
	"flag"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/catatsuy/private-isu/benchmarker/score"
	"github.com/catatsuy/private-isu/benchmarker/worker"
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
	quitC := make(chan bool)
	quit := false
	var mu sync.RWMutex

	workersC := make(chan *worker.Worker, 20)

	go func() {
		for {
			workersC <- worker.NewWorker(target)

			mu.RLock()
			if quit {
				break
			}
			mu.RUnlock()
		}
	}()

	go func() {
		// for stopping goroutines
		<-quitC
		mu.Lock()
		quit = true
		mu.Unlock()
	}()

	toppageNotLogin := worker.NewScenario("GET", "/me")
	toppageNotLogin.ExpectedStatusCode = 200
	toppageNotLogin.ExpectedLocation = "/"
	toppageNotLogin.Checked = true
	toppageNotLogin.CheckFunc = func(w *worker.Worker, body io.Reader) {
		doc, _ := goquery.NewDocumentFromReader(body)

		doc.Find("img").Each(func(_ int, s *goquery.Selection) {
			url, _ := s.Attr("src")
			imgReq := worker.NewScenario("GET", url)
			imgReq.ExpectedStatusCode = 200
			imgReq.Play(w)
		})
	}

	go func() {
		// not login
		for {
			toppageNotLogin.Play(<-workersC)

			mu.RLock()
			if quit {
				break
			}
			mu.RUnlock()
		}
	}()

	login := worker.NewScenario("POST", "/login")
	login.ExpectedStatusCode = 200
	login.ExpectedLocation = "/"

	mepage := worker.NewScenario("GET", "/me")
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

			mu.RLock()
			if quit {
				break
			}
			mu.RUnlock()
		}
	}()

	<-ec
	quitC <- true

	var errs []error

	fmt.Printf("score: %d, suceess: %d, fail: %d\n",
		score.GetInstance().GetScore(),
		score.GetInstance().GetSucesses(),
		score.GetInstance().GetFails(),
	)

	for _, err := range errs {
		fmt.Println(err)
	}

	return ExitCodeOK
}
