package main

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/catatsuy/private-isu/benchmarker/checker"
	"github.com/catatsuy/private-isu/benchmarker/util"
)

func checkHTML(f func(*goquery.Document) error) func(io.Reader) error {
	return func(r io.Reader) error {
		doc, err := goquery.NewDocumentFromReader(r)
		if err != nil {
			return errors.New("ページのHTMLがパースできませんでした")
		}
		return f(doc)
	}
}

// 1ページに表示される画像にリクエストする
// TODO: 画像には並列リクエストするべきでは？
func loadImages(s *checker.Session, imageURLs []string) {
	for _, url := range imageURLs {
		imgReq := checker.NewAssetAction(url, &checker.Asset{})
		imgReq.Description = "投稿画像を読み込めること"
		imgReq.Play(s)
	}
}

func extractImages(doc *goquery.Document) []string {
	imageURLs := []string{}

	doc.Find("img.isu-image").Each(func(_ int, selection *goquery.Selection) {
		if url, ok := selection.Attr("src"); ok {
			imageURLs = append(imageURLs, url)
		}
	}).Length()

	return imageURLs
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
	a := checker.NewAssetAction("/favicon.ico", &checker.Asset{MD5: "ad4b0f606e0f8465bc4c4c170b37e1a3"})
	a.Description = "faviconが読み込めること"
	a.Play(s)

	a = checker.NewAssetAction("js/timeago.min.js", &checker.Asset{MD5: "f2d4c53400d0a46de704f5a97d6d04fb"})
	a.Description = "timeago.min.jsが読み込めること"
	a.Play(s)

	a = checker.NewAssetAction("/js/main.js", &checker.Asset{MD5: "9c309fed7e360c57a705978dab2c68ad"})
	a.Description = "main.jsが読み込めること"
	a.Play(s)

	a = checker.NewAssetAction("/css/style.css", &checker.Asset{MD5: "e4c3606a18d11863189405eb5c6ca551"})
	a.Description = "style.cssが読み込めること"
	a.Play(s)
}

// インデックスにリクエストして「もっと見る」を最大10ページ辿る
// WaitAfterTimeout秒たったら問答無用で打ち切る
func indexMoreAndMoreScenario(s *checker.Session) {
	var imageURLs []string
	start := time.Now()

	imagePerPageChecker := checkHTML(func(doc *goquery.Document) error {
		imageURLs = extractImages(doc)
		if len(imageURLs) < PostsPerPage {
			return errors.New("1ページに表示される画像の数が足りません")
		}
		return nil
	})

	index := checker.NewAction("GET", "/")
	index.ExpectedLocation = `^/$`
	index.Description = "インデックスページが表示できること"
	index.CheckFunc = imagePerPageChecker
	err := index.Play(s)
	if err != nil {
		return
	}

	loadAssets(s)
	loadImages(s, imageURLs)

	offset := util.RandomNumber(10) // 10は適当。URLをバラけさせるため
	for i := 0; i < 10; i++ {       // 10ページ辿る
		maxCreatedAt := time.Date(2016, time.January, 2, 11, 46, 21-PostsPerPage*i+offset, 0, time.FixedZone("Asia/Tokyo", 9*60*60))

		imageURLs = []string{}
		posts := checker.NewAction("GET", "/posts?max_created_at="+url.QueryEscape(maxCreatedAt.Format(time.RFC3339)))
		posts.Description = "インデックスページの「もっと見る」が表示できること"
		posts.CheckFunc = imagePerPageChecker
		err := posts.Play(s)
		if err != nil {
			return
		}

		loadImages(s, imageURLs)

		if time.Since(start) > WaitAfterTimeout {
			break
		}
	}
}

// インデックスページを5回表示するだけ（負荷かける用）
// WaitAfterTimeout秒たったら問答無用で打ち切る
func loadIndexScenario(s *checker.Session) {
	var imageURLs []string
	start := time.Now()

	imagePerPageChecker := checkHTML(func(doc *goquery.Document) error {
		imageURLs = extractImages(doc)
		if len(imageURLs) < PostsPerPage {
			return errors.New("1ページに表示される画像の数が足りません")
		}
		return nil
	})

	index := checker.NewAction("GET", "/")
	index.ExpectedLocation = `^/$`
	index.Description = "インデックスページが表示できること"
	index.CheckFunc = imagePerPageChecker
	err := index.Play(s)
	if err != nil {
		return
	}

	loadAssets(s)
	loadImages(s, imageURLs)

	for i := 0; i < 4; i++ {
		// あとの4回はDOMをパースしない。トップページをキャッシュして超高速に返されたとき対策
		index := checker.NewAction("GET", "/")
		index.ExpectedLocation = `^/$`
		index.Description = "インデックスページが表示できること"
		err := index.Play(s)
		if err != nil {
			return
		}

		loadAssets(s)
		loadImages(s, imageURLs) // 画像は初回と同じものにリクエスト投げる

		if time.Since(start) > WaitAfterTimeout {
			break
		}
	}
}

// /@:account_name のページにアクセスして投稿ページをいくつか開いていく
// WaitAfterTimeout秒たったら問答無用で打ち切る
func userAndPostPageScenario(s *checker.Session, accountName string) {
	var imageURLs []string
	var postLinks []string
	start := time.Now()

	userPage := checker.NewAction("GET", "/@"+accountName)
	userPage.Description = "ユーザーページ"
	userPage.CheckFunc = checkHTML(func(doc *goquery.Document) error {
		imageURLs = extractImages(doc)
		postLinks = extractPostLinks(doc)
		return nil
	})
	err := userPage.Play(s)
	if err != nil {
		return
	}

	loadAssets(s)
	loadImages(s, imageURLs)

	for _, link := range postLinks {
		postPage := checker.NewAction("GET", link)
		postPage.Description = "投稿単体ページが表示できること"
		postPage.CheckFunc = checkHTML(func(doc *goquery.Document) error {
			imageURLs = extractImages(doc)
			if len(imageURLs) < 1 {
				return errors.New("投稿単体ページに投稿画像が表示されていません")
			}
			return nil
		})
		err := postPage.Play(s)
		if err != nil {
			return
		}

		loadAssets(s)
		loadImages(s, imageURLs)

		if time.Since(start) > WaitAfterTimeout {
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
	login.ExpectedLocation = `^/$`
	login.Description = "ログインできること"
	login.PostData = map[string]string{
		"account_name": me.AccountName,
		"password":     me.Password,
	}
	err := login.Play(s)
	if err != nil {
		return
	}

	userPage := checker.NewAction("GET", "/@"+accountName)
	userPage.Description = "ユーザーページが表示できること"
	userPage.CheckFunc = checkHTML(func(doc *goquery.Document) error {

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
	})
	err = userPage.Play(s)
	if err != nil {
		return
	}

	if postID == "" {
		return
	}

	comment := checker.NewAction("POST", "/comment")
	comment.ExpectedLocation = "^/posts/" + postID + "$"
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
	var imageURLs []string
	var ok bool

	login := checker.NewAction("POST", "/login")
	login.ExpectedLocation = `^/$`
	login.Description = "ログインできること"
	login.PostData = map[string]string{
		"account_name": me.AccountName,
		"password":     me.Password,
	}
	login.CheckFunc = checkHTML(func(doc *goquery.Document) error {

		csrfToken, ok = doc.Find(`input[name="csrf_token"]`).First().Attr("value")
		if !ok {
			return errors.New("CSRFトークンが取得できません")
		}

		return nil
	})
	err := login.Play(s)
	if err != nil {
		return
	}

	postImage := checker.NewUploadAction("POST", "/", "file")
	postImage.Description = "画像を投稿してリダイレクトされること"
	postImage.ExpectedLocation = `^/posts/\d+$`
	postImage.Asset = image
	postImage.PostData = map[string]string{
		"body":       sentence,
		"csrf_token": csrfToken,
	}
	postImage.CheckFunc = checkHTML(func(doc *goquery.Document) error {
		imageURLs = extractImages(doc)
		if len(imageURLs) < 1 {
			return errors.New("投稿した画像が表示されていません")
		}
		return nil
	})

	err = postImage.Play(s)
	if err != nil {
		return
	}

	getImage := checker.NewAssetAction(imageURLs[0], image)
	getImage.Description = "投稿した画像と一致すること"
	getImage.Play(s)
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
	login.ExpectedLocation = `^/login$`
	login.PostData = fakeUser
	login.CheckFunc = checkHTML(func(doc *goquery.Document) error {

		message := strings.TrimSpace(doc.Find(`#notice-message`).Text())
		if message != "アカウント名かパスワードが間違っています" {
			return errors.New("ログインエラーメッセージが表示されていません")
		}
		return nil
	})

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
	login.ExpectedLocation = `^/login$`
	login.PostData = fakeUser
	login.CheckFunc = checkHTML(func(doc *goquery.Document) error {

		message := strings.TrimSpace(doc.Find(`#notice-message`).Text())
		if message != "アカウント名かパスワードが間違っています" {
			return errors.New("ログインエラーメッセージが表示されていません")
		}
		return nil
	})

	login.Play(s)
}

// 管理者ユーザーでないなら /admin/banned にアクセスできない
func cannotAccessAdminScenario(s *checker.Session, me user) {
	login := checker.NewAction("POST", "/login")
	login.ExpectedLocation = `^/$`
	login.Description = "Adminユーザーでログインできること"

	login.PostData = map[string]string{
		"account_name": me.AccountName,
		"password":     me.Password,
	}
	err := login.Play(s)
	if err != nil {
		return
	}

	adminPage := checker.NewAction("GET", "/admin/banned")
	adminPage.ExpectedStatusCode = http.StatusForbidden

	adminPage.Play(s)
}

// 間違ったCSRF Tokenで画像を投稿できない
func cannotPostWrongCSRFTokenScenario(s *checker.Session, me user, image *checker.Asset) {
	login := checker.NewAction("POST", "/login")
	login.ExpectedLocation = `^/$`
	login.Description = "正しくログインできること"

	login.PostData = map[string]string{
		"account_name": me.AccountName,
		"password":     me.Password,
	}
	err := login.Play(s)
	if err != nil {
		return
	}

	postImage := checker.NewUploadAction("POST", "/", "file")
	postImage.ExpectedStatusCode = 422
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
	var imageURLs []string

	login := checker.NewAction("POST", "/login")
	login.ExpectedLocation = `^/$`
	login.Description = "ログインするとユーザー名が表示されること"
	login.PostData = map[string]string{
		"account_name": me.AccountName,
		"password":     me.Password,
	}
	login.CheckFunc = checkHTML(func(doc *goquery.Document) error {

		imageURLs = extractImages(doc)

		name := doc.Find(`.isu-account-name`).Text()
		if name == "" {
			return errors.New("ユーザー名が表示されていません")
		} else if name != me.AccountName {
			return errors.New("表示されているユーザー名が正しくありません")
		}
		return nil
	})
	err := login.Play(s)
	if err != nil {
		return
	}

	loadAssets(s)
	loadImages(s, imageURLs) // この画像へのアクセスでSet-Cookieされてたら失敗する

	logout := checker.NewAction("GET", "/logout")
	logout.ExpectedLocation = `^/$`
	logout.Description = "ログアウトするとユーザー名が表示されないこと"
	logout.CheckFunc = checkHTML(func(doc *goquery.Document) error {

		imageURLs = extractImages(doc)

		name := doc.Find(`.isu-account-name`).Text()
		if name != "" {
			return errors.New("ログアウトしてもユーザー名が表示されています")
		}
		return nil
	})
	err = logout.Play(s)
	if err != nil {
		return
	}

	loadAssets(s)
	loadImages(s, imageURLs)
}

// 新規登録→画像投稿→banされる
func banScenario(s1, s2 *checker.Session, u user, admin user, image *checker.Asset, sentence string) {
	var csrfToken string
	var imageURLs []string
	var userID string
	var ok bool
	accountName := util.RandomLUNStr(25)
	password := util.RandomLUNStr(25)

	register := checker.NewAction("POST", "/register")
	register.ExpectedLocation = `^/$`
	register.Description = "新規登録できること"
	register.PostData = map[string]string{
		"account_name": accountName,
		"password":     password,
	}
	register.CheckFunc = checkHTML(func(doc *goquery.Document) error {
		name := doc.Find(`.isu-account-name`).Text()
		if name == "" {
			return errors.New("ユーザー名が表示されていません")
		} else if name != accountName {
			return errors.New("表示されているユーザー名が正しくありません")
		}
		csrfToken, ok = doc.Find(`input[name="csrf_token"]`).First().Attr("value")
		if !ok {
			return errors.New("CSRFトークンが取得できません")
		}

		return nil
	})
	err := register.Play(s1)
	if err != nil {
		return
	}

	postImage := checker.NewUploadAction("POST", "/", "file")
	postImage.Description = "画像を投稿してリダイレクトされること"
	postImage.ExpectedLocation = `^/posts/\d+$`
	postImage.Asset = image
	postImage.PostData = map[string]string{
		"body":       util.RandomLUNStr(15),
		"csrf_token": csrfToken,
	}
	postImage.CheckFunc = checkHTML(func(doc *goquery.Document) error {
		imageURLs = extractImages(doc)
		if len(imageURLs) < 1 {
			return errors.New("投稿した画像が表示されていません")
		}
		return nil
	})

	if len(imageURLs) < 1 {
		return // このケースは上のCheckFuncの中で既にエラーにしてある
	}

	imageURL := imageURLs[0]

	getImage := checker.NewAssetAction(imageURL, image)
	getImage.Description = "投稿した画像と一致することを確認"
	err = getImage.Play(s1)
	if err != nil {
		return
	}

	login := checker.NewAction("POST", "/login")
	login.ExpectedLocation = `^/$`
	login.Description = "管理ユーザーでログインできること"
	login.PostData = map[string]string{
		"account_name": admin.AccountName,
		"password":     admin.Password,
	}
	login.CheckFunc = checkHTML(func(doc *goquery.Document) error {
		imageURLs = extractImages(doc)
		for _, url := range imageURLs {
			if url == imageURL {
				return nil // 投稿した画像が正しく表示されている
			}
		}
		return errors.New("投稿した画像が表示されていません")
	})
	err = login.Play(s2)
	if err != nil {
		return
	}

	banPage := checker.NewAction("GET", "/admin/banned")
	banPage.Description = "管理ユーザーが管理ページにアクセスできること"
	banPage.ExpectedLocation = `^/admin/banned$`
	banPage.CheckFunc = checkHTML(func(doc *goquery.Document) error {
		csrfToken, ok = doc.Find(`input[name="csrf_token"]`).First().Attr("value")
		if !ok {
			return errors.New("CSRFトークンが取得できません")
		}
		userID, ok = doc.Find(`input[data-account-name="` + accountName + `"]`).First().Attr("value")
		if !ok {
			return errors.New("新規登録されたユーザーが管理ページに表示されていません")
		}
		return nil
	})
	err = banPage.Play(s2)
	if err != nil {
		return
	}

	ban := checker.NewAction("POST", "/admin/banned")
	ban.Description = "ユーザーの禁止ができること"
	ban.ExpectedLocation = `^/admin/banned$`
	ban.PostData = map[string]string{
		"uid[]":      userID,
		"csrf_token": csrfToken,
	}
	err = ban.Play(s2)
	if err != nil {
		return
	}

	index := checker.NewAction("GET", "/")
	index.Description = "トップページに禁止ユーザーの画像が表示されていないこと"
	index.CheckFunc = checkHTML(func(doc *goquery.Document) error {
		imageURLs = extractImages(doc)
		for _, url := range imageURLs {
			if url == imageURL {
				return errors.New("禁止ユーザーの画像が表示されています")
			}
		}
		return nil
	})
	index.Play(s2)
}
