package checker

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/catatsuy/private-isu/benchmarker/cache"
)

type Action struct {
	Method string
	Path   string

	PostData           map[string]string
	Headers            map[string]string
	ExpectedStatusCode int
	ExpectedLocation   string
	ExpectedHeaders    map[string]string
	ExpectedHTML       map[string]string

	Description string

	CheckFunc func(body io.Reader) error
}

const (
	suceessGetScore    = 1
	suceessPostScore   = 2
	suceessUploadScore = 5

	failErrorScore     = 10
	failExceptionScore = 20
	failDelayPostScore = 100
)

type Asset struct {
	Path string
	MD5  string
	Type string
}

func NewAction(method, path string) *Action {
	return &Action{
		Method:             method,
		Path:               path,
		ExpectedStatusCode: http.StatusOK,
	}
}

func (a *Action) Play(s *Session) error {
	formData := url.Values{}
	for key, val := range a.PostData {
		formData.Set(key, val)
	}

	buf := bytes.NewBufferString(formData.Encode())
	req, err := s.NewRequest(a.Method, a.Path, buf)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return s.Fail(failExceptionScore, req, errors.New("リクエストに失敗しました (主催者に連絡してください)"))
	}

	for key, val := range a.Headers {
		req.Header.Add(key, val)
	}

	if req.Method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	res, err := s.SendRequest(req)

	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return s.Fail(failExceptionScore, req, errors.New("リクエストがタイムアウトしました"))
		}
		fmt.Fprintln(os.Stderr, err)
		return s.Fail(failExceptionScore, req, errors.New("リクエストに失敗しました"))
	}

	defer res.Body.Close()

	if res.StatusCode != a.ExpectedStatusCode {
		return s.Fail(failErrorScore, res.Request, fmt.Errorf("response code should be %d, got %d", a.ExpectedStatusCode, res.StatusCode))
	}

	if a.ExpectedLocation != "" {
		if !regexp.MustCompile(a.ExpectedLocation).MatchString(res.Request.URL.Path) {
			return s.Fail(
				failErrorScore,
				res.Request,
				fmt.Errorf(
					"リダイレクト先URLが正しくありません: expected '%s', got '%s'",
					a.ExpectedLocation, res.Request.URL.Path,
				))
		}
	}

	if a.CheckFunc != nil {
		err := a.CheckFunc(res.Body)
		if err != nil {
			return s.Fail(
				failErrorScore,
				res.Request,
				err,
			)
		}
	}

	s.Success(suceessGetScore)

	if a.Method == "POST" {
		s.Success(suceessPostScore)
	}

	return nil
}

type AssetAction struct {
	*Action
	Asset *Asset
}

func NewAssetAction(path string, asset *Asset) *AssetAction {
	return &AssetAction{
		Asset: asset,
		Action: &Action{
			Method:             "GET",
			Path:               path,
			ExpectedStatusCode: http.StatusOK,
		},
	}
}

func (a *AssetAction) Play(s *Session) error {
	formData := url.Values{}
	for key, val := range a.PostData {
		formData.Set(key, val)
	}

	buf := bytes.NewBufferString(formData.Encode())
	req, err := s.NewRequest(a.Method, a.Path, buf)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return s.Fail(failExceptionScore, req, errors.New("リクエストに失敗しました (主催者に連絡してください)"))
	}

	for key, val := range a.Headers {
		req.Header.Add(key, val)
	}

	urlCache, cacheFound := cache.GetInstance().Get(a.Path)
	if cacheFound {
		urlCache.Apply(req)
	}

	res, err := s.SendRequest(req)

	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return s.Fail(failExceptionScore, req, errors.New("リクエストがタイムアウトしました"))
		}
		fmt.Fprintln(os.Stderr, err)
		return s.Fail(failExceptionScore, req, errors.New("リクエストに失敗しました"))
	}

	// 2回ioutil.ReadAllを呼ぶとおかしくなる
	uc, md5 := cache.NewURLCache(res)
	if uc != nil {
		cache.GetInstance().Set(a.Path, uc)
		if res.StatusCode == http.StatusOK && a.Asset.MD5 == "" {
			a.Asset.MD5 = md5
		}
	} else if a.Asset.MD5 == "" {
		a.Asset.MD5 = md5
	}

	success := false

	// キャッシュが有効でかつStatusNotModifiedのときは成功
	if cacheFound && res.StatusCode == http.StatusNotModified {
		success = true
	}

	if res.StatusCode == http.StatusOK &&
		((uc == nil && md5 == a.Asset.MD5) || uc != nil) {
		success = true
	}

	defer res.Body.Close()

	if !success {
		return s.Fail(
			failErrorScore,
			res.Request,
			fmt.Errorf("静的ファイルが正しくありません"),
		)
	}

	s.Success(suceessGetScore)

	return nil
}

type UploadAction struct {
	*Action
	UploadParamName string
	Asset           *Asset
}

func NewUploadAction(method, path, uploadParamname string) *UploadAction {
	return &UploadAction{
		UploadParamName: uploadParamname,
		Action: &Action{
			Method:             method,
			Path:               path,
			ExpectedStatusCode: http.StatusOK,
		},
	}
}

func (a *UploadAction) Play(s *Session) error {
	req, err := s.NewFileUploadRequest(a.Path, a.PostData, a.UploadParamName, a.Asset)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return s.Fail(failExceptionScore, req, errors.New("リクエストに失敗しました (主催者に連絡してください)"))
	}

	for key, val := range a.Headers {
		req.Header.Add(key, val)
	}

	res, err := s.SendRequest(req)

	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return s.Fail(failExceptionScore, req, errors.New("リクエストがタイムアウトしました"))
		}
		fmt.Fprintln(os.Stderr, err)
		return s.Fail(failExceptionScore, req, errors.New("リクエストに失敗しました"))
	}

	defer res.Body.Close()

	if res.StatusCode != a.ExpectedStatusCode {
		return s.Fail(
			failErrorScore,
			res.Request,
			fmt.Errorf("ステータスコードが正しくありません: expected %d, got %d", a.ExpectedStatusCode, res.StatusCode),
		)
	}

	if a.ExpectedLocation != "" {
		if !regexp.MustCompile(a.ExpectedLocation).MatchString(res.Request.URL.Path) {
			return s.Fail(
				failErrorScore,
				res.Request,
				fmt.Errorf(
					"リダイレクト先URLが正しくありません: expected '%s', got '%s'",
					a.ExpectedLocation, res.Request.URL.Path,
				))
		}
	}

	if a.CheckFunc != nil {
		err := a.CheckFunc(res.Body)
		if err != nil {
			return s.Fail(
				failErrorScore,
				res.Request,
				err,
			)
		}
	}

	s.Success(suceessUploadScore)
	s.Success(suceessGetScore)

	return nil
}
