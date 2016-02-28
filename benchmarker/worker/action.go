package worker

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
	for key, val := range s.PostData {
		formData.Set(key, val)
	}

	buf := bytes.NewBufferString(formData.Encode())
	req, err := w.NewRequest(s.Method, s.Path, buf)

	if err != nil {
		return w.Fail(req, err)
	}

	for key, val := range s.Headers {
		req.Header.Add(key, val)
	}

	if req.Method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	res, err := w.SendRequest(req)

	if err != nil {
		return w.Fail(req, err)
	}

	defer res.Body.Close()

	if res.StatusCode != s.ExpectedStatusCode {
		w.Fail(res.Request, fmt.Errorf("Response code should be %d, got %d", s.ExpectedStatusCode, res.StatusCode))
	}

	if s.ExpectedLocation != "" {
		if s.ExpectedLocation != res.Request.URL.Path {
			return w.Fail(
				res.Request,
				fmt.Errorf(
					"Expected location is miss match %s, got: %s",
					s.ExpectedLocation, res.Request.URL.Path,
				))
		}
	}

	if s.CheckFunc != nil {
		err := s.CheckFunc(w, res.Body)
		if err != nil {
			return w.Fail(
				res.Request,
				err,
			)
		}
	}

	w.Success(1)

	return nil
}

func (a *Action) PlayWithImage(s *Session) error {
	formData := url.Values{}
	for key, val := range s.PostData {
		formData.Set(key, val)
	}

	buf := bytes.NewBufferString(formData.Encode())
	req, err := w.NewRequest(s.Method, s.Path, buf)

	if err != nil {
		return w.Fail(req, err)
	}

	for key, val := range s.Headers {
		req.Header.Add(key, val)
	}

	urlCache, found := cache.GetInstance().Get(s.Path)
	if found {
		urlCache.Apply(req)
	}

	res, err := w.SendRequest(req)

	if err != nil {
		return w.Fail(req, err)
	}

	// 2回ioutil.ReadAllを呼ぶとおかしくなる
	uc, md5 := cache.NewURLCache(res)
	if uc != nil {
		cache.GetInstance().Set(s.Path, uc)
	}

	success := false

	if res.StatusCode == http.StatusNotModified {
		success = true
	}

	if res.StatusCode == http.StatusOK &&
		((uc == nil && util.GetMD5ByIO(res.Body) == s.Asset.MD5) || md5 == s.Asset.MD5) {
		success = true
	}

	defer res.Body.Close()

	if !success {
		return w.Fail(
			res.Request,
			fmt.Errorf(
				"Expected location is miss match %s, got: %s",
				s.ExpectedLocation, res.Request.URL.Path,
			))
	}

	w.Success(1)

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
