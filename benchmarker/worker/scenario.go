package worker

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
)

type Scenario struct {
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

	Checked   bool
	CheckFunc func(w *Worker, body io.Reader) error
}

type Asset struct {
	Path string
	MD5  string
}

func NewScenario(method, path string) *Scenario {
	return &Scenario{
		Method: method,
		Path:   path,

		ExpectedStatusCode: 200,

		Checked: false,
	}
}

func (s *Scenario) Play(w *Worker) error {
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

	res, err := w.SendRequest(req, false)

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

	if s.Checked {
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

func (s *Scenario) PlayWithFile(w *Worker, paramName string) error {
	req, err := w.NewFileUploadRequest(s.Path, s.PostData, paramName, s.Asset.Path)

	if err != nil {
		return w.Fail(req, err)
	}

	for key, val := range s.Headers {
		req.Header.Add(key, val)
	}

	res, err := w.SendRequest(req, false)

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

	if s.Checked {
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
