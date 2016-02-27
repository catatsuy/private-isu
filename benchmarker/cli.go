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

	worker.SetTargetHost(target)

	type user struct {
		AccountName string
		Password    string
	}
	users := []*user{
		&user{
			AccountName: "user1",
			Password:    "user1user1",
		},
		&user{
			AccountName: "user2",
			Password:    "user2user2",
		},
		&user{
			AccountName: "user3",
			Password:    "user3user3",
		},
		&user{
			AccountName: "user4",
			Password:    "user4user4",
		},
		&user{
			AccountName: "user5",
			Password:    "user5user5",
		},
	}

	images := []*worker.Asset{
		&worker.Asset{
			MD5:  "Cb0e066UYAAwxtT.jpg",
			Path: "./userdata/img/8c4d0286cc2c92b418cb6b20fa2055d4",
		},
		&worker.Asset{
			MD5:  "8c4d0286cc2c92b418cb6b20fa2055d4",
			Path: "./userdata/img/Cb0e066UYAAwxtT.jpg",
		},
		&worker.Asset{
			MD5:  "e43267883243c297d8f6f66582fc098b",
			Path: "./userdata/img/Cb0rChYUUAAERl8.jpg",
		},
		&worker.Asset{
			MD5:  "623176077a8da7cc7602c132cb91deeb",
			Path: "./userdata/img/Cb5XdejUcAA78Nz.jpg",
		},
		&worker.Asset{
			MD5:  "45d7ba976202a85a90e17282d7f7a781",
			Path: "./userdata/img/CbJLMlcUcAER_Sg.jpg",
		},
		&worker.Asset{
			MD5:  "de906699516c228eee7f025d3e88057b",
			Path: "./userdata/img/CbOuZvjUEAA5r0K.jpg",
		},
		&worker.Asset{
			MD5:  "b50e41b163b501f1aa3cada9a21696c4",
			Path: "./userdata/img/CbT1pABVAAA1OMG.jpg",
		},
		&worker.Asset{
			MD5:  "aa7929fb4ec357063e12701226d0fa3d",
			Path: "./userdata/img/Cba_gezUMAApMPw.jpg",
		},
		&worker.Asset{
			MD5:  "a36c35c8db3e32bde24f9e77d811fecb",
			Path: "./userdata/img/CbyvdPtUcAMiasE.jpg",
		},
		&worker.Asset{
			MD5:  "5985d209ba9d3fe9c0ded4fdbf4cdeb5",
			Path: "./userdata/img/CcCJ26eVAAAf9sh.jpg",
		},
		&worker.Asset{
			MD5:  "c5e0fb9d1132ed936813c07c480730b9",
			Path: "./userdata/img/CcJYpMDUUAA2xXc.jpg",
		},
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
			workersC <- worker.NewWorker()
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
			imgReq.Asset = &worker.Asset{}
			imgReq.PlayWithImage(w)
			if exit > 15 {
				return false
			} else {
				exit += 1
				return true
			}
		})

		return nil
	}

	// TOPページに非ログイン状態でひたすらアクセス
	// 画像にもリクエストを送っている
	go func() {
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

	// ログインしてmypageをちゃんと見れるか確認
	go func() {
		for {
			u := users[util.RandomNumber(len(users))]
			login.PostData = map[string]string{
				"account_name": u.AccountName,
				"password":     u.Password,
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
		postTopImg.PlayWithPostFile(w, "file")
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

	// ログインして、画像を投稿して、mypageをcheckして、コメントを投稿
	go func() {
		for {
			u := users[util.RandomNumber(len(users))]
			login.PostData = map[string]string{
				"account_name": u.AccountName,
				"password":     u.Password,
			}
			postTopImg.Asset = images[util.RandomNumber(len(images))]
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

	// ユーザーを作って、ログインして画像を投稿する
	// そのユーザーはBAN機能を使って消される
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

	fmt.Printf("score: %d, suceess: %d, fail: %d\n",
		score.GetInstance().GetScore(),
		score.GetInstance().GetSucesses(),
		score.GetInstance().GetFails(),
	)

	for _, err := range worker.GetFails() {
		fmt.Println(err)
	}

	return ExitCodeOK
}
