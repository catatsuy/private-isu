package main

import (
	"net/url"
	"time"

	"io"

	"errors"

	"strings"

	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/catatsuy/private-isu/benchmarker/checker"
	"github.com/catatsuy/private-isu/benchmarker/util"
)

// 1ページに表示される画像にリクエストする
// TODO: 画像には並列リクエストするべきでは？
func loadImages(s *checker.Session, imageUrls []string) {
	for _, url := range imageUrls {
		imgReq := checker.NewAssetAction(url, &checker.Asset{})
		imgReq.Description = "投稿画像を読み込めること"
		imgReq.Play(s)
	}
}

func extractImages(doc *goquery.Document) []string {
	imageUrls := []string{}

	doc.Find("img.isu-image").Each(func(_ int, selection *goquery.Selection) {
		if url, ok := selection.Attr("src"); ok {
			imageUrls = append(imageUrls, url)
		}
	}).Length()

	return imageUrls
}

func extractPostLinks(doc *goquery.Document) []string {
	postLinks := []string{}

	doc.Find("a.isu-post-permalink").Each(func(_ int, selection *goquery.Selection) {
		if url, ok := selection.Attr("href"); ok {
			postLinks = append(postLinks, url)
		}
	}).Length()

	return postLinks
}

// 普通のページに表示されるべき静的ファイルに一通りアクセス
func loadAssets(s *checker.Session) {
	a := checker.NewAssetAction("/favicon.ico", &checker.Asset{})
	a.ExpectedLocation = "/favicon.ico"
	a.Description = "faviconが読み込めること"
	a.Play(s)

	a = checker.NewAssetAction("js/jquery-2.2.0.js", &checker.Asset{})
	a.ExpectedLocation = "js/jquery-2.2.0.js"
	a.Description = "jqueryが読み込めること"
	a.Play(s)

	a = checker.NewAssetAction("js/jquery.timeago.js", &checker.Asset{})
	a.ExpectedLocation = "js/jquery.timeago.js"
	a.Description = "jquery.timeago.jsが読み込めること"
	a.Play(s)

	a = checker.NewAssetAction("js/jquery.timeago.ja.js", &checker.Asset{})
	a.ExpectedLocation = "js/jquery.timeago.ja.js"
	a.Description = "jquery.timeago.ja.jsが読み込めること"
	a.Play(s)

	a = checker.NewAssetAction("/js/main.js", &checker.Asset{})
	a.ExpectedLocation = "/js/main.js"
	a.Description = "main.jsが読み込めること"
	a.Play(s)

	a = checker.NewAssetAction("/css/style.css", &checker.Asset{})
	a.ExpectedLocation = "/css/style.css"
	a.Description = "style.cssが読み込めること"
	a.Play(s)
}

// インデックスにリクエストして「もっと見る」を最大10ページ辿る
// WaitAfterTimeout秒たったら問答無用で打ち切る
func indexMoreAndMoreScenario(s *checker.Session) {
	var imageUrls []string
	start := time.Now()

	imagePerPageChecker := func(s *checker.Session, body io.Reader) error {
		doc, err := goquery.NewDocumentFromReader(body)
		if err != nil {
			return errors.New("ページが正しく読み込めませんでした")
		}
		imageUrls = extractImages(doc)
		if len(imageUrls) < PostsPerPage {
			return errors.New("1ページに表示される画像の数が足りません")
		}
		return nil
	}

	index := checker.NewAction("GET", "/")
	index.ExpectedLocation = "/"
	index.Description = "インデックスページが表示できること"
	index.CheckFunc = imagePerPageChecker
	index.Play(s)

	loadAssets(s)
	loadImages(s, imageUrls)

	offset := util.RandomNumber(10) // 10は適当。URLをバラけさせるため
	for i := 0; i < 10; i++ {       // 10ページ辿る
		maxCreatedAt := time.Date(2016, time.January, 2, 11, 46, 21-PostsPerPage*i+offset, 0, time.FixedZone("Asia/Tokyo", 9*60*60))

		imageUrls = []string{}
		posts := checker.NewAction("GET", "/posts?max_created_at="+url.QueryEscape(maxCreatedAt.Format(time.RFC3339)))
		posts.Description = "インデックスページの「もっと見る」が表示できること"
		posts.CheckFunc = imagePerPageChecker
		posts.Play(s)

		loadImages(s, imageUrls)

		if time.Now().Sub(start) > WaitAfterTimeout {
			break
		}
	}
}

// インデックスページを5回表示するだけ（負荷かける用）
// WaitAfterTimeout秒たったら問答無用で打ち切る
func loadIndexScenario(s *checker.Session) {
	var imageUrls []string
	start := time.Now()

	imagePerPageChecker := func(s *checker.Session, body io.Reader) error {
		doc, err := goquery.NewDocumentFromReader(body)
		if err != nil {
			return errors.New("ページが正しく読み込めませんでした")
		}
		imageUrls = extractImages(doc)
		if len(imageUrls) < PostsPerPage {
			return errors.New("1ページに表示される画像の数が足りません")
		}
		return nil
	}

	index := checker.NewAction("GET", "/")
	index.ExpectedLocation = "/"
	index.Description = "インデックスページが表示できること"
	index.CheckFunc = imagePerPageChecker
	index.Play(s)

	loadAssets(s)
	loadImages(s, imageUrls)

	for i := 0; i < 4; i++ {
		// あとの4回はDOMをパースしない。トップページをキャッシュして超高速に返されたとき対策
		index := checker.NewAction("GET", "/")
		index.ExpectedLocation = "/"
		index.Description = "インデックスページが表示できること"

		loadAssets(s)
		loadImages(s, imageUrls) // 画像は初回と同じものにリクエスト投げる

		if time.Now().Sub(start) > WaitAfterTimeout {
			break
		}
	}
}

// /@:account_name のページにアクセスして投稿ページをいくつか開いていく
// WaitAfterTimeout秒たったら問答無用で打ち切る
func userAndPostPageScenario(s *checker.Session, accountName string) {
	var imageUrls []string
	var postLinks []string
	start := time.Now()

	userPage := checker.NewAction("GET", "/@"+accountName)
	userPage.Description = "ユーザーページ"
	userPage.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, err := goquery.NewDocumentFromReader(body)
		if err != nil {
			return errors.New("ページが正しく読み込めませんでした")
		}
		imageUrls = extractImages(doc)
		postLinks = extractPostLinks(doc)
		return nil
	}
	userPage.Play(s)

	loadAssets(s)
	loadImages(s, imageUrls)

	for _, link := range postLinks {
		postPage := checker.NewAction("GET", link)
		postPage.Description = "投稿単体ページが表示できること"
		postPage.CheckFunc = func(s *checker.Session, body io.Reader) error {
			doc, err := goquery.NewDocumentFromReader(body)
			if err != nil {
				return errors.New("ページが正しく読み込めませんでした")
			}
			imageUrls = extractImages(doc)
			if len(imageUrls) < 1 {
				return errors.New("投稿単体ページに投稿画像が表示されていません")
			}
			return nil
		}
		postPage.Play(s)

		loadAssets(s)
		loadImages(s, imageUrls)

		if time.Now().Sub(start) > WaitAfterTimeout {
			break
		}
	}
}

// ログインして /@:account_name のページにアクセスして一番上の投稿にコメントする
// 簡略化のために画像や静的ファイルへのアクセスはスキップする
func commentScenario(s *checker.Session, me user, accountName string, sentence string) {
	var csrfToken string
	var postID string
	var ok bool

	login := checker.NewAction("POST", "/login")
	login.ExpectedLocation = "/"
	login.Description = "ログインできること"
	login.PostData = map[string]string{
		"account_name": me.AccountName,
		"password":     me.Password,
	}
	login.Play(s)

	userPage := checker.NewAction("GET", "/@"+accountName)
	userPage.Description = "ユーザーページが表示できること"
	userPage.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, err := goquery.NewDocumentFromReader(body)
		if err != nil {
			return errors.New("ページが正しく読み込めませんでした")
		}

		sel := doc.Find(`div.isu-post`).First()

		if sel.Length() == 0 {
			return nil // 1枚も投稿が無いユーザー
		}

		csrfToken, ok = sel.Find(`input[name="csrf_token"]`).First().Attr("value")
		if !ok {
			return errors.New("CSRFトークンが取得できません")
		}

		postID, ok = sel.Find(`input[name="post_id"]`).First().Attr("value")
		if !ok {
			return errors.New("post_idが取得できません")
		}

		return nil
	}
	userPage.Play(s)

	if postID == "" {
		return
	}

	comment := checker.NewAction("POST", "/comment")
	comment.ExpectedLocation = "/posts/" + postID
	comment.PostData = map[string]string{
		"post_id":    postID,
		"comment":    sentence,
		"csrf_token": csrfToken,
	}
	comment.Play(s)
}

// ログインして画像を投稿する
// 簡略化のために画像や静的ファイルへのアクセスはスキップする
func postImageScenario(s *checker.Session, me user, image *checker.Asset, sentence string) {
	var csrfToken string
	var imageUrls []string
	var ok bool
	var err error

	login := checker.NewAction("POST", "/login")
	login.ExpectedLocation = "/"
	login.Description = "ログインできること"
	login.PostData = map[string]string{
		"account_name": me.AccountName,
		"password":     me.Password,
	}
	login.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, err := goquery.NewDocumentFromReader(body)
		if err != nil {
			return errors.New("ページが正しく読み込めませんでした")
		}

		csrfToken, ok = doc.Find(`input[name="csrf_token"]`).First().Attr("value")
		if !ok {
			return errors.New("CSRFトークンが取得できません")
		}

		return nil
	}
	login.Play(s)

	postImage := checker.NewUploadAction("POST", "/", "file")
	postImage.Description = "画像を投稿してリダイレクトされること"
	postImage.Asset = image
	postImage.PostData = map[string]string{
		"body":       sentence,
		"csrf_token": csrfToken,
		"type":       "image/jpeg", // TODO: pngやgifもあるのでどうにかする
	}

	postImage.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, err := goquery.NewDocumentFromReader(body)
		if err != nil {
			return errors.New("ページが正しく読み込めませんでした")
		}
		imageUrls = extractImages(doc)
		if len(imageUrls) < 1 {
			return errors.New("投稿した画像が表示されていません")
		}
		return nil
	}

	_, err = postImage.PlayWithURL(s)
	if err != nil {
		return // TODO: どういうエラーハンドリングが適切か考える
	}

	if len(imageUrls) < 1 {
		return // このケースは上のCheckFuncの中で既にエラーにしてある
	}

	getImage := checker.NewAssetAction(imageUrls[0], image)
	getImage.Description = "投稿した画像と一致することを確認"
}

// 適当なユーザー名でログインしようとする
// ログインできないことをチェック
func cannotLoginNonexistentUserScenario(s *checker.Session) {
	fakeAccountName := util.RandomLUNStr(util.RandomNumber(15) + 10)
	fakeUser := map[string]string{
		"account_name": fakeAccountName,
		"password":     fakeAccountName,
	}

	login := checker.NewAction("POST", "/login")
	login.Description = "存在しないユーザー名でログインできないこと"
	login.ExpectedLocation = "/login"
	login.PostData = fakeUser
	login.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		message := strings.TrimSpace(doc.Find(`#notice-message`).Text())
		if message != "アカウント名かパスワードが間違っています" {
			return errors.New("ログインエラーメッセージが表示されていません")
		}
		return nil
	}

	login.Play(s)
}

// 誤ったパスワードでログインできない
func cannotLoginWrongPasswordScenario(s *checker.Session, me user) {
	fakeUser := map[string]string{
		"account_name": me.AccountName,
		"password":     util.RandomLUNStr(util.RandomNumber(15) + 10),
	}

	login := checker.NewAction("POST", "/login")
	login.Description = "間違ったパスワードでログインできないこと"
	login.ExpectedLocation = "/login"
	login.PostData = fakeUser
	login.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, _ := goquery.NewDocumentFromReader(body)

		message := strings.TrimSpace(doc.Find(`#notice-message`).Text())
		if message != "アカウント名かパスワードが間違っています" {
			return errors.New("ログインエラーメッセージが表示されていません")
		}
		return nil
	}

	login.Play(s)
}

// 管理者ユーザーでないなら /admin/banned にアクセスできない
func cannotAccessAdminScenario(s *checker.Session, me user) {
	login := checker.NewAction("POST", "/login")
	login.ExpectedLocation = "/"
	login.Description = "Adminユーザーでログインできること"

	login.PostData = map[string]string{
		"account_name": me.AccountName,
		"password":     me.Password,
	}
	login.Play(s)

	a := checker.NewAction("GET", "/admin/banned")
	a.ExpectedStatusCode = http.StatusForbidden

	a.Play(s)
}

// 間違ったCSRF Tokenで画像を投稿できない
func cannotPostWrongCSRFTokenScenario(s *checker.Session, me user, image *checker.Asset) {
	login := checker.NewAction("POST", "/login")
	login.ExpectedLocation = "/"
	login.Description = "正しくログインできること"

	login.PostData = map[string]string{
		"account_name": me.AccountName,
		"password":     me.Password,
	}
	login.Play(s)

	postImage := checker.NewUploadAction("POST", "/", "file")
	postImage.ExpectedStatusCode = http.StatusForbidden
	postImage.Description = "間違ったCSRFトークンでは画像を投稿できないこと"
	postImage.Asset = image
	postImage.PostData = map[string]string{
		"body":       util.RandomLUNStr(25),
		"csrf_token": util.RandomLUNStr(64),
	}
	postImage.Play(s)
}

// ログインすると右上にアカウント名が出て、ログインしないとアカウント名が出ない
// 画像のキャッシュにSet-Cookieを含んでいた場合、/にアカウント名が含まれる
func loginScenario(s *checker.Session, me user) {
	var imageUrls []string

	login := checker.NewAction("POST", "/login")
	login.ExpectedLocation = "/"
	login.Description = "ログインするとユーザー名が表示されること"
	login.PostData = map[string]string{
		"account_name": me.AccountName,
		"password":     me.Password,
	}
	login.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, err := goquery.NewDocumentFromReader(body)
		if err != nil {
			return errors.New("ページが正しく読み込めませんでした")
		}

		imageUrls = extractImages(doc)

		name := doc.Find(`.isu-account-name`).Text()
		if name == "" {
			return errors.New("ユーザー名が表示されていません")
		} else if name != me.AccountName {
			return errors.New("表示されているユーザー名が正しくありません")
		}
		return nil
	}
	login.Play(s)

	// TODO: ここまででfailであればこれ以降は進めない

	loadAssets(s)
	loadImages(s, imageUrls) // この画像へのアクセスでSet-Cookieされてたら失敗する

	logout := checker.NewAction("GET", "/logout")
	logout.ExpectedLocation = "/"
	logout.Description = "ログアウトするとユーザー名が表示されないこと"
	logout.CheckFunc = func(s *checker.Session, body io.Reader) error {
		doc, err := goquery.NewDocumentFromReader(body)
		if err != nil {
			return errors.New("ページが正しく読み込めませんでした")
		}

		imageUrls = extractImages(doc)

		name := doc.Find(`.isu-account-name`).Text()
		if name != "" {
			return errors.New("ログアウトしてもユーザー名が表示されています")
		}
		return nil
	}
	logout.Play(s)

	loadAssets(s)
	loadImages(s, imageUrls)
}
