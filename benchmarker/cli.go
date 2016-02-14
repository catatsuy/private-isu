package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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
	toppageNotLogin.CheckFunc = func(w *worker.Worker, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		exit := 0
		doc.Find("img").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			url, _ := s.Attr("src")
			imgReq := worker.NewScenario("GET", url)
			imgReq.ExpectedStatusCode = 200
			imgReq.Play(w)
			if exit > 15 {
				return false
			} else {
				exit += 1
				return true
			}
		})

		return nil
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

	postTopImg := worker.NewScenario("POST", "/")
	postTopImg.ExpectedStatusCode = 200
	postTopImg.ExpectedLocation = "/"

	mepageCheck := worker.NewScenario("GET", "/me")
	mepageCheck.ExpectedStatusCode = 200
	mepageCheck.Checked = true

	mepageCheck.CheckFunc = func(w *worker.Worker, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		url, _ := doc.Find(`img`).First().Attr("src")
		imgReq := worker.NewScenario("GET", url)
		imgReq.ExpectedStatusCode = 200
		imgReq.Checked = true
		imgReq.CheckFunc = func(w *worker.Worker, body io.Reader) error {
			if getMD5ByIO(body) == postTopImg.Asset.MD5 {
				return nil
			} else {
				return fmt.Errorf("Error")
			}
		}
		imgReq.Play(w)

		return nil
	}

	getIndexAfterPostImg := worker.NewScenario("GET", "/")
	getIndexAfterPostImg.ExpectedStatusCode = 200
	getIndexAfterPostImg.Checked = true

	getIndexAfterPostImg.CheckFunc = func(w *worker.Worker, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		token, _ := doc.Find(`input[name="csrf_token"]`).First().Attr("value")
		postTopImg.PostData = map[string]string{
			"body":       "aaaaaaaaa",
			"csrf_token": token,
			"type":       "image/jpeg",
		}
		postTopImg.Asset = &worker.Asset{
			Path: "./userdata/img/data.jpg",
			MD5:  "a5243f84e4859a9647ecc508239a9a51",
		}
		postTopImg.PlayWithFile(w, "file")
		mepageCheck.Play(w)

		return nil
	}

	go func() {
		for {
			login.PostData = map[string]string{
				"account_name": "catatsuy",
				"password":     "kaneko",
			}
			w := <-workersC
			login.Play(w)
			getIndexAfterPostImg.Play(w)

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

func getMD5(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}

func getMD5ByIO(r io.Reader) string {
	bytes, _ := ioutil.ReadAll(r)
	return getMD5(bytes)
}
