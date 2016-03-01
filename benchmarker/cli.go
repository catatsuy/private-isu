package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/catatsuy/private-isu/benchmarker/checker"
	"github.com/catatsuy/private-isu/benchmarker/score"
	"github.com/catatsuy/private-isu/benchmarker/util"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota

	FailThreshold    = 5
	BenchmarkTimeout = 30 * time.Second
	SessionQueueSize = 20
)

// CLI is the command line object
type CLI struct {
	// outStream and errStream are the stdout and stderr
	// to write message from the CLI.
	outStream, errStream io.Writer
}

type user struct {
	AccountName string
	Password    string
}

var quit bool
var quitLock sync.RWMutex

// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	quit = false
	var (
		target string

		version bool
		debug   bool
	)

	// Define option flag parse
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)

	flags.StringVar(&target, "target", "", "")
	flags.StringVar(&target, "t", "", "(Short)")

	flags.BoolVar(&version, "version", false, "Print version information and quit.")

	flags.BoolVar(&debug, "debug", false, "Debug mode")
	flags.BoolVar(&debug, "d", false, "Debug mode")

	// Parse commandline flag
	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeError
	}

	// Show version
	if version {
		fmt.Fprintf(cli.errStream, "%s version %s\n", Name, Version)
		return ExitCodeOK
	}

	terr := checker.SetTargetHost(target)
	if terr != nil {
		fmt.Println(terr.Error())
		return ExitCodeError
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
	adminUsers := []*user{
		&user{
			AccountName: "adminuser1",
			Password:    "adminuser1",
		},
		&user{
			AccountName: "adminuser2",
			Password:    "adminuser2",
		},
		&user{
			AccountName: "adminuser3",
			Password:    "adminuser3",
		},
		&user{
			AccountName: "adminuser4",
			Password:    "adminuser4",
		},
	}

	images := []*checker.Asset{
		&checker.Asset{
			MD5:  "8c4d0286cc2c92b418cb6b20fa2055d4",
			Path: "./userdata/img/Cb0e066UYAAwxtT.jpg",
		},
		&checker.Asset{
			MD5:  "e43267883243c297d8f6f66582fc098b",
			Path: "./userdata/img/Cb0rChYUUAAERl8.jpg",
		},
		&checker.Asset{
			MD5:  "623176077a8da7cc7602c132cb91deeb",
			Path: "./userdata/img/Cb5XdejUcAA78Nz.jpg",
		},
		&checker.Asset{
			MD5:  "45d7ba976202a85a90e17282d7f7a781",
			Path: "./userdata/img/CbJLMlcUcAER_Sg.jpg",
		},
		&checker.Asset{
			MD5:  "de906699516c228eee7f025d3e88057b",
			Path: "./userdata/img/CbOuZvjUEAA5r0K.jpg",
		},
		&checker.Asset{
			MD5:  "b50e41b163b501f1aa3cada9a21696c4",
			Path: "./userdata/img/CbT1pABVAAA1OMG.jpg",
		},
		&checker.Asset{
			MD5:  "aa7929fb4ec357063e12701226d0fa3d",
			Path: "./userdata/img/Cba_gezUMAApMPw.jpg",
		},
		&checker.Asset{
			MD5:  "a36c35c8db3e32bde24f9e77d811fecb",
			Path: "./userdata/img/CbyvdPtUcAMiasE.jpg",
		},
		&checker.Asset{
			MD5:  "5985d209ba9d3fe9c0ded4fdbf4cdeb5",
			Path: "./userdata/img/CcCJ26eVAAAf9sh.jpg",
		},
		&checker.Asset{
			MD5:  "c5e0fb9d1132ed936813c07c480730b9",
			Path: "./userdata/img/CcJYpMDUUAA2xXc.jpg",
		},
	}

	timeUp := time.After(BenchmarkTimeout)
	done := make(chan bool)

	sessionsQueue := make(chan *checker.Session, SessionQueueSize)

	setupSessionGenrator(sessionsQueue, done)

	setupWorkerToppageNotLogin(sessionsQueue)

	setupWorkerMypageCheck(sessionsQueue, users)
	setupWorkerPostData(sessionsQueue, users, images)
	setupWorkerBanUser(sessionsQueue, images, adminUsers)

	<-timeUp

	quitLock.Lock()
	quit = true
	quitLock.Unlock()

	<-done
	close(sessionsQueue)

	fmt.Printf("score: %d, suceess: %d, fail: %d\n",
		score.GetInstance().GetScore(),
		score.GetInstance().GetSucesses(),
		score.GetInstance().GetFails(),
	)

	if !debug {
		// 通常は適当にsortしてuniqしたログを出す
		for _, err := range score.GetFailErrors() {
			fmt.Println(err.Error())
		}
	} else {
		// debugモードなら生ログを出力
		for _, err := range score.GetFailRawErrors() {
			fmt.Println(err.Error())
		}
	}

	// Failが多い場合はステータスコードを非0にする
	if score.GetInstance().GetFails() >= FailThreshold {
		return ExitCodeError
	}

	return ExitCodeOK
}

func setupSessionGenrator(sessionsQueue chan *checker.Session, done chan bool) {
	// sessionsQueueにsessionを用意しておく
	// キューとして使って並列度が高くなりすぎないようにするのと、
	// 時間が来たらcloseする
	go func() {
		for {
			sessionsQueue <- checker.NewSession()
			quitLock.RLock()
			if quit {
				quitLock.RUnlock()
				done <- true
				break
			}
			quitLock.RUnlock()
		}
	}()
}

func setupWorkerToppageNotLogin(sessionsQueue chan *checker.Session) {
	toppageNotLogin := genActionToppageNotLogin()
	mypageNotLogin := genActionMypageNotLogin()

	go func() {
		for {
			s := <-sessionsQueue
			// /にログインせずにアクセスして、画像にリクエストを送る
			// その後、同じセッションを使い回して/mypageにアクセス
			// 画像のキャッシュにSet-Cookieを含んでいた場合、/mypageのリダイレクト先でfailする
			toppageNotLogin.Play(s)
			mypageNotLogin.Play(s)
		}
	}()
}

// 非ログインで/mypageにアクセスして/にリダイレクトするかチェック
func genActionMypageNotLogin() *checker.Action {
	a := checker.NewAction("GET", "/mypage")
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "/"
	a.Description = "/mypageは非ログイン時に/にリダイレクトがかかる"

	return a
}

// TOPページに非ログイン状態でひたすらアクセス
// 画像にもリクエストを送っている
func genActionToppageNotLogin() *checker.Action {
	a := checker.NewAction("GET", "/")
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "/"
	a.Description = "/にある画像にひたすらアクセス"
	a.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		imageRequestCount := 0
		maxImageRequest := 15
		doc.Find("img").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
			url, _ := selection.Attr("src")
			imgReq := checker.NewAssetAction("GET", url)
			imgReq.ExpectedStatusCode = http.StatusOK
			imgReq.Asset = &checker.Asset{}
			imgReq.Play(s)
			if imageRequestCount > maxImageRequest {
				return false
			} else {
				imageRequestCount += 1
				return true
			}
		})

		return nil
	}

	return a
}

// ログインしてmypageをちゃんと見れるか確認
func setupWorkerMypageCheck(sessionsQueue chan *checker.Session, users []*user) {
	login := genActionLogin()
	mypage := genActionMypage()

	go func() {
		for {
			u := users[util.RandomNumber(len(users))]
			login.PostData = map[string]string{
				"account_name": u.AccountName,
				"password":     u.Password,
			}
			s := <-sessionsQueue
			login.Play(s)
			mypage.Play(s)
		}
	}()
}

func genActionLogin() *checker.Action {
	a := checker.NewAction("POST", "/login")
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "/"
	a.Description = "ログイン"

	return a
}

func genActionMypage() *checker.Action {
	a := checker.NewAction("GET", "/mypage")
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "/mypage"
	a.Description = "ログインして、/mypageに"

	return a
}

func genActionPostTopImg() *checker.Action {
	a := checker.NewAction("POST", "/")
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "/"
	a.Description = "画像を投稿"

	return a
}

func genActionCheckMypage() *checker.Action {
	a := checker.NewAction("GET", "/mypage")
	a.ExpectedStatusCode = http.StatusOK

	a.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		url, _ := doc.Find(`img`).First().Attr("src")
		imgReq := checker.NewAssetAction("GET", url)
		imgReq.ExpectedStatusCode = http.StatusOK
		imgReq.Asset = &checker.Asset{}
		imgReq.Play(s)

		return nil
	}

	return a
}

func genActionPostComment() *checker.Action {
	a := checker.NewAction("POST", "/comment")
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "/"

	return a
}

func genActionGetIndexAfterPostImg(postTopImg *checker.Action, checkMypage *checker.Action) *checker.Action {
	a := checker.NewAction("GET", "/")
	a.ExpectedStatusCode = http.StatusOK

	a.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		token, _ := doc.Find(`input[name="csrf_token"]`).First().Attr("value")
		postTopImg.PostData = map[string]string{
			"body":       util.RandomLUNStr(util.RandomNumber(20) + 10),
			"csrf_token": token,
			"type":       "image/jpeg",
		}
		postTopImg.PlayWithPostFile(s, "file")
		checkMypage.Play(s)

		return nil
	}

	return a
}

func genActionGetIndexAfterPostComment(postComment *checker.Action) *checker.Action {
	a := checker.NewAction("GET", "/")
	a.ExpectedStatusCode = http.StatusOK

	a.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		token, _ := doc.Find(`input[name="csrf_token"]`).First().Attr("value")
		postID, _ := doc.Find(`input[name="post_id"]`).First().Attr("value")
		postComment.PostData = map[string]string{
			"post_id":    postID,
			"comment":    "comment",
			"csrf_token": token,
		}
		postComment.Play(s)

		return nil
	}
	return a
}

func setupWorkerPostData(sessionsQueue chan *checker.Session, users []*user, images []*checker.Asset) {
	login := genActionLogin()
	postTopImg := genActionPostTopImg()

	checkMypage := genActionCheckMypage()
	getIndexAfterPostImg := genActionGetIndexAfterPostImg(postTopImg, checkMypage)

	postComment := genActionPostComment()
	getIndexAfterPostComment := genActionGetIndexAfterPostComment(postComment)

	// ログインして、画像を投稿して、mypageをcheckして、コメントを投稿
	go func() {
		for {
			u := users[util.RandomNumber(len(users))]
			login.PostData = map[string]string{
				"account_name": u.AccountName,
				"password":     u.Password,
			}
			postTopImg.Asset = images[util.RandomNumber(len(images))]
			s := <-sessionsQueue
			login.Play(s)
			getIndexAfterPostImg.Play(s)
			getIndexAfterPostComment.Play(s)
		}
	}()
}

func genActionPostRegister() *checker.Action {
	a := checker.NewAction("POST", "/register")
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "/"
	return a
}

func genActionBanUser(accountName string) *checker.Action {
	a := checker.NewAction("GET", "/admin/banned")
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "/admin/banned"
	a.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)
		token, _ := doc.Find(`input[name="csrf_token"]`).First().Attr("value")
		uid, _ := doc.Find(`input[data-account-name="` + accountName + `"]`).First().Attr("value")

		postAdminBanned := checker.NewAction("POST", "/admin/banned")
		postAdminBanned.ExpectedStatusCode = http.StatusOK
		postAdminBanned.ExpectedLocation = "/admin/banned"
		postAdminBanned.PostData = map[string]string{
			"uid[]":      uid,
			"csrf_token": token,
		}
		postAdminBanned.Play(s)

		return nil
	}

	return a
}

func genActionCheckBannedUser(targetUserAccountName string) *checker.Action {
	a := checker.NewAction("GET", "/")
	a.ExpectedStatusCode = http.StatusOK

	a.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		imageRequestCount := 0
		maxImageRequest := 20
		existErr := false

		doc.Find(`.isu-post-account-name`).EachWithBreak(func(_ int, selection *goquery.Selection) bool {
			accountName := strings.TrimSpace(selection.Text())
			if accountName == targetUserAccountName {
				existErr = true
				return false
			}
			if imageRequestCount > maxImageRequest {
				return false
			} else {
				imageRequestCount += 1
				return true
			}
			return true
		})

		if existErr {
			return fmt.Errorf("BANされたユーザーの投稿が表示されています")
		}

		return nil
	}

	return a
}

func setupWorkerBanUser(sessionsQueue chan *checker.Session, images []*checker.Asset, adminUsers []*user) {
	interval := time.Tick(10 * time.Second)

	login := genActionLogin()
	postRegister := genActionPostRegister()
	postTopImg := genActionPostTopImg()
	checkMypage := genActionCheckMypage()
	getIndexAfterPostImg := genActionGetIndexAfterPostImg(postTopImg, checkMypage)

	// ユーザーを作って、ログインして画像を投稿する
	// そのユーザーはBAN機能を使って消される
	go func() {
		for {
			s1 := <-sessionsQueue

			targetUserAccountName := util.RandomLUNStr(25)
			deletedUser := map[string]string{
				"account_name": targetUserAccountName,
				"password":     targetUserAccountName,
			}

			postRegister.PostData = deletedUser
			postRegister.Play(s1)
			login.PostData = deletedUser
			login.Play(s1)
			postTopImg.Asset = images[util.RandomNumber(len(images))]
			getIndexAfterPostImg.Play(s1)
			postTopImg.PlayWithPostFile(s1, "file")

			u := adminUsers[util.RandomNumber(len(adminUsers))]
			login.PostData = map[string]string{
				"account_name": u.AccountName,
				"password":     u.Password,
			}
			s2 := <-sessionsQueue
			login.Play(s2)

			banUser := genActionBanUser(targetUserAccountName)
			banUser.Play(s2)

			checkBanned := genActionCheckBannedUser(targetUserAccountName)
			checkBanned.Play(s2)
			<-interval
		}
	}()
}
