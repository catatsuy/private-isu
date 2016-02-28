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

	PostData map[string]string
	Headers  map[string]string
	Asset    *Asset

	ExpectedStatusCode int
	ExpectedLocation   string
	ExpectedHeaders    map[string]string
	ExpectedAssets     map[string]string
	ExpectedHTML       map[string]string

	Description string

	CheckFunc func(w *Session, body io.Reader) error
}

type Asset struct {
	Path string
	MD5  string
}

func NewAction(method, path string) *Action {
	return &Action{
		Method: method,
		Path:   path,

		ExpectedStatusCode: 200,
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
		return s.Fail(req, err)
	}

	for key, val := range a.Headers {
		req.Header.Add(key, val)
	}

	if req.Method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	res, err := s.SendRequest(req)

	if err != nil {
		return s.Fail(req, err)
	}

	defer res.Body.Close()

	if res.StatusCode != a.ExpectedStatusCode {
		s.Fail(res.Request, fmt.Errorf("Response code should be %d, got %d", a.ExpectedStatusCode, res.StatusCode))
	}

	if a.ExpectedLocation != "" {
		if a.ExpectedLocation != res.Request.URL.Path {
			return s.Fail(
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
				res.Request,
				err,
			)
		}
	}

	s.Success(1)

	return nil
}

func (a *Action) PlayWithImage(s *Session) error {
	formData := url.Values{}
	for key, val := range a.PostData {
		formData.Set(key, val)
	}

	buf := bytes.NewBufferString(formData.Encode())
	req, err := s.NewRequest(a.Method, a.Path, buf)

	if err != nil {
		return s.Fail(req, err)
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
		return s.Fail(req, err)
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
			res.Request,
			fmt.Errorf(
				"Expected location is miss match %s, got: %s",
				a.ExpectedLocation, res.Request.URL.Path,
			))
	}

	s.Success(1)

	return nil
}

func (a *Action) PlayWithPostFile(s *Session, paramName string) error {
	req, err := s.NewFileUploadRequest(a.Path, a.PostData, paramName, a.Asset.Path)

	if err != nil {
		return s.Fail(req, err)
	}

	for key, val := range a.Headers {
		req.Header.Add(key, val)
	}

	res, err := s.SendRequest(req)

	if err != nil {
		return s.Fail(req, err)
	}

	defer res.Body.Close()

	if res.StatusCode != a.ExpectedStatusCode {
		s.Fail(res.Request, fmt.Errorf("Response code should be %d, got %d", a.ExpectedStatusCode, res.StatusCode))
	}

	if a.ExpectedLocation != "" {
		if a.ExpectedLocation != res.Request.URL.Path {
			return s.Fail(
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
				res.Request,
				err,
			)
		}
	}

	s.Success(1)

	return nil
}
