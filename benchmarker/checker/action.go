package checker

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/catatsuy/private-isu/benchmarker/cache"
	"github.com/catatsuy/private-isu/benchmarker/util"
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

	CheckFunc func(w *Session, body io.Reader) error
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
		return s.Fail(failExceptionScore, req, err)
	}

	for key, val := range a.Headers {
		req.Header.Add(key, val)
	}

	if req.Method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	res, err := s.SendRequest(req)

	if err != nil {
		return s.Fail(failExceptionScore, req, err)
	}

	defer res.Body.Close()

	if res.StatusCode != a.ExpectedStatusCode {
		return s.Fail(failErrorScore, res.Request, fmt.Errorf("Response code should be %d, got %d", a.ExpectedStatusCode, res.StatusCode))
	}

	if a.ExpectedLocation != "" {
		if a.ExpectedLocation != res.Request.URL.Path {
			return s.Fail(
				failErrorScore,
				res.Request,
				fmt.Errorf(
					"Expected location is miss match %s, got: %s",
					a.ExpectedLocation, res.Request.URL.Path,
				))
		}
	}

	if a.CheckFunc != nil {
		err := a.CheckFunc(s, res.Body)
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
		return s.Fail(failExceptionScore, req, err)
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
		return s.Fail(failExceptionScore, req, err)
	}

	// 2回ioutil.ReadAllを呼ぶとおかしくなる
	uc, md5 := cache.NewURLCache(res)
	if uc != nil {
		cache.GetInstance().Set(a.Path, uc)
	}

	success := false

	// キャッシュが有効でかつStatusNotModifiedのときは成功
	if cacheFound && res.StatusCode == http.StatusNotModified {
		success = true
	}

	if res.StatusCode == http.StatusOK &&
		((uc == nil && util.GetMD5ByIO(res.Body) == a.Asset.MD5) || md5 == a.Asset.MD5) {
		success = true
	}

	defer res.Body.Close()

	if !success {
		return s.Fail(
			failErrorScore,
			res.Request,
			fmt.Errorf(
				"Expected location is miss match %s, got: %s",
				a.ExpectedLocation, res.Request.URL.Path,
			))
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
		return s.Fail(failExceptionScore, req, err)
	}

	for key, val := range a.Headers {
		req.Header.Add(key, val)
	}

	res, err := s.SendRequest(req)

	if err != nil {
		return s.Fail(failExceptionScore, req, err)
	}

	defer res.Body.Close()

	if res.StatusCode != a.ExpectedStatusCode {
		return s.Fail(failErrorScore, res.Request, fmt.Errorf("Response code should be %d, got %d", a.ExpectedStatusCode, res.StatusCode))
	}

	if a.ExpectedLocation != "" {
		if a.ExpectedLocation != res.Request.URL.Path {
			return s.Fail(
				failErrorScore,
				res.Request,
				fmt.Errorf(
					"Expected location is miss match %s, got: %s",
					a.ExpectedLocation, res.Request.URL.Path,
				))
		}
	}

	if a.CheckFunc != nil {
		err := a.CheckFunc(s, res.Body)
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

func (a *UploadAction) PlayWithURL(s *Session) (string, error) {
	req, err := s.NewFileUploadRequest(a.Path, a.PostData, a.UploadParamName, a.Asset)

	if err != nil {
		return "", s.Fail(failExceptionScore, req, err)
	}

	for key, val := range a.Headers {
		req.Header.Add(key, val)
	}

	res, err := s.SendRequest(req)

	if err != nil {
		return "", s.Fail(failExceptionScore, req, err)
	}

	defer res.Body.Close()

	if res.StatusCode != a.ExpectedStatusCode {
		return "", s.Fail(failErrorScore, res.Request, fmt.Errorf("Response code should be %d, got %d", a.ExpectedStatusCode, res.StatusCode))
	}

	if a.ExpectedLocation != "" {
		if a.ExpectedLocation != res.Request.URL.Path {
			return "", s.Fail(
				failErrorScore,
				res.Request,
				fmt.Errorf(
					"Expected location is miss match %s, got: %s",
					a.ExpectedLocation, res.Request.URL.Path,
				))
		}
	}

	if a.CheckFunc != nil {
		err := a.CheckFunc(s, res.Body)
		if err != nil {
			return "", s.Fail(
				failErrorScore,
				res.Request,
				err,
			)
		}
	}

	s.Success(suceessUploadScore)
	s.Success(suceessGetScore)

	return res.Request.URL.Path, nil
}
