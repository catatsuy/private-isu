package main

import (
	"net/url"
	"time"

	"io"

	"errors"

	"github.com/PuerkitoBio/goquery"
	"github.com/catatsuy/private-isu/benchmarker/checker"
	"github.com/catatsuy/private-isu/benchmarker/util"
)

// 1ページに表示される画像にリクエストする
// TODO: 画像には並列リクエストするべきでは？
func loadImages(s *checker.Session, imageUrls []string) {
	for _, url := range imageUrls {
		imgReq := checker.NewAssetAction(url, &checker.Asset{})
		imgReq.Description = "投稿画像"
		imgReq.Play(s)
	}
}

func extractImages(body io.Reader) ([]string, error) {
	imageUrls := []string{}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, errors.New("ページが正しく読み込めませんでした")
	}

	doc.Find("img.isu-image").Each(func(_ int, selection *goquery.Selection) {
		url, _ := selection.Attr("src")
		imageUrls = append(imageUrls, url)
	}).Length()

	return imageUrls, nil
}

// 普通のページに表示されるべき静的ファイルに一通りアクセス
func loadAssets(s *checker.Session) {
	a := checker.NewAssetAction("/favicon.ico", &checker.Asset{})
	a.ExpectedLocation = "/favicon.ico"
	a.Description = "favicon.ico"
	a.Play(s)

	a = checker.NewAssetAction("js/jquery-2.2.0.js", &checker.Asset{})
	a.ExpectedLocation = "js/jquery-2.2.0.js"
	a.Description = "js/jquery-2.2.0.js"
	a.Play(s)

	a = checker.NewAssetAction("js/jquery.timeago.js", &checker.Asset{})
	a.ExpectedLocation = "js/jquery.timeago.js"
	a.Description = "js/jquery.timeago.js"
	a.Play(s)

	a = checker.NewAssetAction("js/jquery.timeago.ja.js", &checker.Asset{})
	a.ExpectedLocation = "js/jquery.timeago.ja.js"
	a.Description = "js/jquery.timeago.ja.js"
	a.Play(s)

	a = checker.NewAssetAction("/js/main.js", &checker.Asset{})
	a.ExpectedLocation = "/js/main.js"
	a.Description = "js/main.js"
	a.Play(s)

	a = checker.NewAssetAction("/css/style.css", &checker.Asset{})
	a.ExpectedLocation = "/css/style.css"
	a.Description = "/css/style.css"
	a.Play(s)
}

// インデックスにリクエストして「もっと見る」を最大10ページ辿る
// 15秒たったら問答無用で打ち切る
func indexMoreAndMoreScenario(s *checker.Session) {
	var imageUrls []string
	var err error
	now := time.Now()

	imagePerPageChecker := func(s *checker.Session, body io.Reader) error {
		imageUrls, err = extractImages(body)
		if err != nil {
			return err
		}
		if len(imageUrls) < PostsPerPage {
			return errors.New("1ページに表示される画像の数が足りません")
		}
		return nil
	}

	index := checker.NewAction("GET", "/")
	index.ExpectedLocation = "/"
	index.Description = "インデックスページ"
	index.CheckFunc = imagePerPageChecker
	index.Play(s)

	loadAssets(s)
	loadImages(s, imageUrls)

	offset := util.RandomNumber(10) // 10は適当。URLをバラけさせるため
	for i := 0; i < 10; i++ {       // 10ページ辿る
		maxCreatedAt := time.Date(2016, time.January, 2, 11, 46, 21-PostsPerPage*i+offset, 0, time.FixedZone("Asia/Tokyo", 9*60*60))

		imageUrls = []string{}
		posts := checker.NewAction("GET", "/posts?max_created_at="+url.QueryEscape(maxCreatedAt.Format(time.RFC3339)))
		posts.Description = "インデックスページの「もっと見る」"
		posts.CheckFunc = imagePerPageChecker
		posts.Play(s)

		loadImages(s, imageUrls)

		if time.Now().Sub(now) > 15*time.Second {
			break
		}
	}
}

// インデックスページを5回表示するだけ（負荷かける用）
// 15秒たったら問答無用で打ち切る
func loadIndexScenario(s *checker.Session) {
	var imageUrls []string
	var err error
	now := time.Now()

	imagePerPageChecker := func(s *checker.Session, body io.Reader) error {
		imageUrls, err = extractImages(body)
		if err != nil {
			return err
		}
		if len(imageUrls) < PostsPerPage {
			return errors.New("1ページに表示される画像の数が足りません")
		}
		return nil
	}

	index := checker.NewAction("GET", "/")
	index.ExpectedLocation = "/"
	index.Description = "インデックスページ"
	index.CheckFunc = imagePerPageChecker
	index.Play(s)

	loadAssets(s)
	loadImages(s, imageUrls)

	for i := 0; i < 4; i++ {
		// あとの4回はDOMをパースしない。トップページをキャッシュして超高速に返されたとき対策
		index := checker.NewAction("GET", "/")
		index.ExpectedLocation = "/"
		index.Description = "インデックスページ"

		loadAssets(s)
		loadImages(s, imageUrls) // 画像は初回と同じものにリクエスト投げる

		if time.Now().Sub(now) > 15*time.Second {
			break
		}
	}
}
