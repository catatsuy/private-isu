package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

	FailThreshold          = 5
	InitializeTimeout      = time.Duration(10) * time.Second
	BenchmarkTimeout       = 30 * time.Second
	DetailedCheckQueueSize = 2
	PostsCheckQueueSize    = 2

	PostsPerPage = 20
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

// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	var (
		target   string
		userdata string

		version bool
		debug   bool
	)

	// Define option flag parse
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)

	flags.StringVar(&target, "target", "", "")
	flags.StringVar(&target, "t", "", "(Short)")

	flags.StringVar(&userdata, "userdata", "", "userdata directory")
	flags.StringVar(&userdata, "u", "", "userdata directory")

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

	targetHost, terr := checker.SetTargetHost(target)
	if terr != nil {
		fmt.Println(terr.Error())
		return ExitCodeError
	}

	initialize := make(chan bool)

	setupInitialize(targetHost, initialize)

	users, adminUsers, sentences, images, err := prepareUserdata(userdata)
	if err != nil {
		fmt.Println(err.Error())
		return ExitCodeError
	}

	<-initialize

	// 最初にDOMチェックなどをやってしまい、通らなければさっさと失敗させる
	detailedCheck(users, adminUsers, sentences, images)

	if score.GetInstance().GetFails() > 0 {
		for _, err := range score.GetFailErrors() {
			fmt.Println(err.Error())
		}
		return ExitCodeError
	}

	postsCheckCh := make(chan bool, PostsCheckQueueSize)
	for i := 0; i < PostsCheckQueueSize; i++ {
		postsCheckCh <- true
	}
	detailedCheckCh := make(chan bool, DetailedCheckQueueSize)
	for i := 0; i < DetailedCheckQueueSize; i++ {
		detailedCheckCh <- true
	}

	timeoutCh := time.After(BenchmarkTimeout)

L:
	for {
		select {
		case <-postsCheckCh:
			go func() {
				checkPostsMoreAndMore(checker.NewSession())
				postsCheckCh <- true
			}()
		case <-detailedCheckCh:
			go func() {
				detailedCheck(users, adminUsers, sentences, images)
				detailedCheckCh <- true
			}()
		case <-timeoutCh:
			break L
		}
	}

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

func prepareUserdata(userdata string) ([]user, []user, []string, []*checker.Asset, error) {
	if userdata == "" {
		return nil, nil, nil, nil, errors.New("userdataディレクトリが指定されていません")
	}
	info, err := os.Stat(userdata)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if !info.IsDir() {
		return nil, nil, nil, nil, errors.New("userdataがディレクトリではありません")
	}

	file, err := os.Open(userdata + "/names.txt")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer file.Close()

	users := []user{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		name := scanner.Text()
		users = append(users, user{AccountName: name, Password: name + name})
	}
	adminUsers := users[:10]

	sentenceFile, err := os.Open(userdata + "/kaomoji.txt")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer sentenceFile.Close()

	sentences := []string{}

	sScanner := bufio.NewScanner(sentenceFile)
	for sScanner.Scan() {
		sentence := sScanner.Text()
		sentences = append(sentences, sentence)
	}

	imgs, err := filepath.Glob(userdata + "/img/000*") // 00001.jpg, 00002.png, 00003.gif など
	if err != nil {
		return nil, nil, nil, nil, err
	}

	images := []*checker.Asset{}

	for _, img := range imgs {
		data, err := ioutil.ReadFile(img)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		images = append(images, &checker.Asset{
			MD5:  util.GetMD5(data),
			Path: img,
		})
	}

	return users, adminUsers, sentences, images, err
}

func checkUserpageNotLogin(s *checker.Session, users []user) {
	userpageNotLogin := genActionUserpageNotLogin(users[util.RandomNumber(len(users))].AccountName)
	userpageNotLogin.Play(s)
}

// 非ログインで/@:account_nameにアクセスして、画像にリクエストを送る
func genActionUserpageNotLogin(accountName string) *checker.Action {
	a := checker.NewAction("GET", "/@"+accountName)
	a.ExpectedStatusCode = http.StatusOK
	a.Description = "非ログインで/@:account_nameにアクセスして、画像にリクエストを送る"
	a.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		doc.Find(`.isu-post`).Each(func(_ int, selection *goquery.Selection) {
			url, _ := selection.Find(".isu-post-image img").Attr("src")
			imgReq := checker.NewAssetAction(url, &checker.Asset{})
			imgReq.Play(s)
		})

		return nil
	}

	return a
}

// /にログインせずにアクセスして、画像にリクエストを送る
// その後、同じセッションを使い回して/にアクセス
// 画像のキャッシュにSet-Cookieを含んでいた場合、/にアカウント名が含まれる
func checkToppageNotLogin(s *checker.Session) {

	indexAndImagesNotLogin := genActionIndexAndImagesNotLogin()
	indexNotLogin := genActionIndexNotLogin()

	indexAndImagesNotLogin.Play(s)
	indexNotLogin.Play(s)
}

// 非ログインで/にアクセスして、ユーザー名が出ていないことを確認
func genActionIndexNotLogin() *checker.Action {
	a := checker.NewAction("GET", "/")
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "/"
	a.Description = "非ログインで/にアクセスしてログイン状態になっていないことを確認"
	a.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		accountName := doc.Find(`.isu-account-name`).Text()
		if accountName == "" {
			return nil
		} else {
			return fmt.Errorf("非ログインユーザーがログインしています")
		}
	}

	return a
}

// TOPページに非ログイン状態でひたすらアクセス
// 画像にもリクエストを送っている
func genActionIndexAndImagesNotLogin() *checker.Action {
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
			imgReq := checker.NewAssetAction(url, &checker.Asset{})
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

func genActionLogin() *checker.Action {
	a := checker.NewAction("POST", "/login")
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "/"
	a.Description = "ログイン"

	return a
}

func genActionPostTopImg() *checker.UploadAction {
	a := checker.NewUploadAction("POST", "/", "file")
	a.ExpectedStatusCode = http.StatusOK
	a.Description = "画像を投稿"

	return a
}

// /posts/:id にリクエストを飛ばして画像のURLを見る
// その画像のURLにリクエストを飛ばして画像が一致しているか確認
func genActionGetPostPageImg(url string, image *checker.Asset) *checker.Action {
	a := checker.NewAction("GET", url)
	a.ExpectedStatusCode = http.StatusOK

	a.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		url, _ := doc.Find(`img`).First().Attr("src")
		imgReq := checker.NewAssetAction(url, image)
		imgReq.Play(s)

		return nil
	}

	return a
}

func genActionPostComment(url, postID, comment, accountName, csrfToken string) *checker.Action {
	a := checker.NewAction("POST", "/comment")
	a.ExpectedLocation = url
	a.ExpectedStatusCode = http.StatusOK
	a.PostData = map[string]string{
		"post_id":    postID,
		"comment":    comment,
		"csrf_token": csrfToken,
	}

	a.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		success := false

		doc.Find(".isu-comment").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
			c := selection.Find(".isu-comment-text").Text()
			an := selection.Find(".isu-comment-account-name").Text()

			if c == comment && an == accountName {
				success = true
				return false
			}

			return true
		})

		if success {
			return nil
		} else {
			return fmt.Errorf("投稿したコメントが表示されていません")
		}
	}

	return a
}

func genActionGetIndexAfterPostImg(postTopImg *checker.UploadAction, accountName string, sentence1 string, sentence2 string) *checker.Action {
	re := regexp.MustCompile("/posts/([0-9]+)")

	a := checker.NewAction("GET", "/")
	a.ExpectedStatusCode = http.StatusOK
	a.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		token, _ := doc.Find(`input[name="csrf_token"]`).First().Attr("value")
		postTopImg.PostData = map[string]string{
			"body":       sentence1,
			"csrf_token": token,
			"type":       "image/jpeg",
		}
		redirectedURL, _ := postTopImg.PlayWithURL(s)
		result := re.FindStringSubmatch(redirectedURL)
		if len(result) < 2 {
			return fmt.Errorf("POSTした後のredirect先が誤っています")
		}

		getPostPageImg := genActionGetPostPageImg(redirectedURL, postTopImg.Asset)
		getPostPageImg.Play(s)

		postComment := genActionPostComment(redirectedURL, result[1], sentence2, accountName, token)
		postComment.Play(s)

		return nil
	}

	return a
}

// ログインして、画像を投稿して、投稿単体ページを確認して、コメントを投稿
func checkPostData(s *checker.Session, users []user, sentences []string, images []*checker.Asset) {
	login := genActionLogin()
	postTopImg := genActionPostTopImg()

	u := users[util.RandomNumber(len(users))]
	login.PostData = map[string]string{
		"account_name": u.AccountName,
		"password":     u.Password,
	}
	postTopImg.Asset = images[util.RandomNumber(len(images))]
	login.Play(s)
	sentence1 := sentences[util.RandomNumber(len(sentences))] + sentences[util.RandomNumber(len(sentences))]
	sentence2 := sentences[util.RandomNumber(len(sentences))] + sentences[util.RandomNumber(len(sentences))]
	getIndexAfterPostImg := genActionGetIndexAfterPostImg(postTopImg, u.AccountName, sentence1, sentence2)
	getIndexAfterPostImg.Play(s)
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
		})

		if existErr {
			return fmt.Errorf("BANされたユーザーの投稿が表示されています")
		}

		return nil
	}

	return a
}

// ユーザーを作って、ログインして画像を投稿する
// そのユーザーはBAN機能を使って消される
func checkBanUser(s1 *checker.Session, s2 *checker.Session, sentences []string, images []*checker.Asset, adminUsers []user) {
	login := genActionLogin()
	postRegister := genActionPostRegister()
	postTopImg := genActionPostTopImg()

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
	sentence1 := sentences[util.RandomNumber(len(sentences))] + sentences[util.RandomNumber(len(sentences))]
	sentence2 := sentences[util.RandomNumber(len(sentences))] + sentences[util.RandomNumber(len(sentences))]
	getIndexAfterPostImg := genActionGetIndexAfterPostImg(postTopImg, targetUserAccountName, sentence1, sentence2)
	getIndexAfterPostImg.Play(s1)
	postTopImg.Play(s1)

	u := adminUsers[util.RandomNumber(len(adminUsers))]
	login.PostData = map[string]string{
		"account_name": u.AccountName,
		"password":     u.Password,
	}

	login.Play(s2)

	banUser := genActionBanUser(targetUserAccountName)
	banUser.Play(s2)

	checkBanned := genActionCheckBannedUser(targetUserAccountName)
	checkBanned.Play(s2)
}

func genActionAppleTouchIconCheck() *checker.Action {
	a := checker.NewAction("GET", "/apple-touch-icon-precomposed.png")
	a.ExpectedStatusCode = http.StatusNotFound
	a.Description = "apple-touch-icon-precomposed.png should not exist"

	return a
}

func genActionFaviconCheck() *checker.AssetAction {
	a := checker.NewAssetAction("/favicon.ico", &checker.Asset{})
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "/favicon.ico"
	a.Description = "favicon.ico"

	return a
}

func genActionJsMainFileCheck() *checker.AssetAction {
	a := checker.NewAssetAction("/js/main.js", &checker.Asset{})
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "/js/main.js"
	a.Description = "js/main.js"

	return a
}

func genActionJsJqueryFileCheck() *checker.AssetAction {
	a := checker.NewAssetAction("js/jquery-2.2.0.js", &checker.Asset{})
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "js/jquery-2.2.0.js"
	a.Description = "js/jquery-2.2.0.js"

	return a
}

func genActionCssFileCheck() *checker.AssetAction {
	a := checker.NewAssetAction("/css/style.css", &checker.Asset{})
	a.ExpectedStatusCode = http.StatusOK
	a.ExpectedLocation = "/css/style.css"
	a.Description = "/css/style.css"

	return a
}

func checkStaticFiles(s *checker.Session) {
	faviconCheck := genActionFaviconCheck()
	appleIconCheck := genActionAppleTouchIconCheck()
	jsMainFileCheck := genActionJsMainFileCheck()
	jsJQueryFileCheck := genActionJsJqueryFileCheck()
	cssFileCheck := genActionCssFileCheck()

	faviconCheck.Play(s)
	appleIconCheck.Play(s)
	jsJQueryFileCheck.Play(s)
	jsMainFileCheck.Play(s)
	cssFileCheck.Play(s)
}

func setupInitialize(targetHost string, initialize chan bool) {
	go func(targetHost string) {
		client := &http.Client{
			Timeout: InitializeTimeout,
		}

		parsedURL, _ := url.Parse("/initialize")
		parsedURL.Scheme = "http"
		parsedURL.Host = targetHost

		res, err := client.Get(parsedURL.String())
		if err != nil {
			initialize <- false
			return
		}
		defer res.Body.Close()
		initialize <- true
	}(targetHost)
}

func detailedCheck(users []user, adminUsers []user, sentences []string, images []*checker.Asset) {
	checkToppageNotLogin(checker.NewSession())
	checkStaticFiles(checker.NewSession())
	checkUserpageNotLogin(checker.NewSession(), users)
	checkPostData(checker.NewSession(), users, sentences, images)
	checkBanUser(checker.NewSession(), checker.NewSession(), sentences, images, adminUsers)
}

func genActionPostsCheck(maxCreatedAt time.Time) *checker.Action {
	a := checker.NewAction("GET", "/posts?max_created_at="+url.QueryEscape(maxCreatedAt.Format(time.RFC3339)))
	a.Description = "もっと見るをひたすら辿っていく"
	a.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		imgCnt := doc.Find("img").Each(func(_ int, selection *goquery.Selection) {
			url, _ := selection.Attr("src")
			imgReq := checker.NewAction("GET", url)
			imgReq.Play(s)
		}).Length()

		if imgCnt < PostsPerPage {
			return errors.New("1ページに表示される画像の数が足りません")
		}
		return nil
	}

	return a
}

// ひらすらトップページの「もっと見る」を開いていく君
func checkPostsMoreAndMore(s *checker.Session) {
	offset := util.RandomNumber(10) // 10は適当。URLをバラけさせるため
	for i := 0; i < 10; i++ {       // 10ページ辿る
		maxCreatedAt := time.Date(2016, time.January, 2, 11, 46, 21-PostsPerPage*i+offset, 0, time.FixedZone("Asia/Tokyo", 9*60*60))
		postsCheck := genActionPostsCheck(maxCreatedAt)
		postsCheck.Play(s)
	}
}
