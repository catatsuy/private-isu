package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/catatsuy/private-isu/benchmarker/score"
	"github.com/catatsuy/private-isu/benchmarker/util"
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

	timeUp := time.After(30 * time.Second)
	done := make(chan bool)
	quit := false
	var mu sync.RWMutex

	workersC := make(chan *worker.Worker, 20)

	// workersCにworkerを用意しておく
	// キューとして使って並列度が高くなりすぎないようにするのと、
	// 時間が来たらcloseする
	go func() {
		for {
			workersC <- worker.NewWorker(target)
			mu.RLock()
			if quit {
				done <- true
				break
			}
			mu.RUnlock()
		}
	}()

	toppageNotLogin := worker.NewScenario("GET", "/mypage")
	toppageNotLogin.ExpectedStatusCode = 200
	toppageNotLogin.ExpectedLocation = "/"
	toppageNotLogin.Description = "/mypageは非ログイン時に/にリダイレクトがかかる"
	toppageNotLogin.Checked = true
	toppageNotLogin.CheckFunc = func(w *worker.Worker, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		exit := 0
		doc.Find("img").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			url, _ := s.Attr("src")
			imgReq := worker.NewScenario("GET", url)
			imgReq.ExpectedStatusCode = 200
			imgReq.PlayWithCached(w)
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
		}
	}()

	login := worker.NewScenario("POST", "/login")
	login.ExpectedStatusCode = 200
	login.ExpectedLocation = "/"
	login.Description = "ログイン"

	mypage := worker.NewScenario("GET", "/mypage")
	mypage.ExpectedStatusCode = 200
	mypage.ExpectedLocation = "/mypage"
	mypage.Description = "ログインして、/mypageに"

	go func() {
		for {
			login.PostData = map[string]string{
				"account_name": "catatsuy",
				"password":     "kaneko",
			}
			w := <-workersC
			login.Play(w)
			mypage.Play(w)
		}
	}()

	postTopImg := worker.NewScenario("POST", "/")
	postTopImg.ExpectedStatusCode = 200
	postTopImg.ExpectedLocation = "/"
	postTopImg.Description = "画像を投稿"

	mypageCheck := worker.NewScenario("GET", "/mypage")
	mypageCheck.ExpectedStatusCode = 200
	mypageCheck.Checked = true

	mypageCheck.CheckFunc = func(w *worker.Worker, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		url, _ := doc.Find(`img`).First().Attr("src")
		imgReq := worker.NewScenario("GET", url)
		imgReq.ExpectedStatusCode = 200
		imgReq.Checked = true
		imgReq.CheckFunc = func(w *worker.Worker, body io.Reader) error {
			if util.GetMD5ByIO(body) == postTopImg.Asset.MD5 {
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
		mypageCheck.Play(w)

		return nil
	}

	postComment := worker.NewScenario("POST", "/comment")
	postComment.ExpectedStatusCode = 200
	postComment.ExpectedLocation = "/"

	getIndexAfterPostComment := worker.NewScenario("GET", "/")
	getIndexAfterPostComment.ExpectedStatusCode = 200
	getIndexAfterPostComment.Checked = true

	getIndexAfterPostComment.CheckFunc = func(w *worker.Worker, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		token, _ := doc.Find(`input[name="csrf_token"]`).First().Attr("value")
		postID, _ := doc.Find(`input[name="post_id"]`).First().Attr("value")
		postComment.PostData = map[string]string{
			"post_id":    postID,
			"comment":    "comment",
			"csrf_token": token,
		}
		postComment.Play(w)

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
			getIndexAfterPostComment.Play(w)
		}
	}()

	getAdminBanned := worker.NewScenario("GET", "/admin/banned")
	getAdminBanned.ExpectedStatusCode = 200
	getAdminBanned.ExpectedLocation = "/admin/banned"
	getAdminBanned.Checked = true
	getAdminBanned.CheckFunc = func(w *worker.Worker, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)
		token, _ := doc.Find(`input[name="csrf_token"]`).First().Attr("value")

		postAdminBanned := worker.NewScenario("POST", "/admin/banned")
		postAdminBanned.ExpectedStatusCode = 200
		postAdminBanned.ExpectedLocation = "/admin/banned"
		postAdminBanned.PostData = map[string]string{
			"uid[]":      "11",
			"csrf_token": token,
		}
		postAdminBanned.Play(w)

		return nil
	}

	checkBanned := worker.NewScenario("GET", "/")
	checkBanned.ExpectedStatusCode = 200
	checkBanned.Checked = true

	checkBanned.CheckFunc = func(w *worker.Worker, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		exit := 0
		existErr := false

		doc.Find(`.isu-post-account-name`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
			account_name := strings.TrimSpace(s.Text())
			if account_name == "banned_user" {
				existErr = true
				return false
			}
			if exit > 20 {
				return false
			} else {
				exit += 1
				return true
			}
			return true
		})

		if existErr {
			return errors.New("BANされたユーザーの投稿が表示されています")
		}

		return nil
	}

	interval := time.Tick(10 * time.Second)

	go func() {
		for {
			<-interval

			login.PostData = map[string]string{
				"account_name": "catatsuy",
				"password":     "kaneko",
			}
			w := <-workersC
			login.Play(w)
			getAdminBanned.Play(w)
			checkBanned.Play(w)
		}
	}()

	<-timeUp

	mu.Lock()
	quit = true
	mu.Unlock()

	<-done
	close(workersC)

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
