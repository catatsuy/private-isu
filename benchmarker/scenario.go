package main

import (
	"bytes"
	"io/ioutil"
	"net/url"
)

type Scenario struct {
	Method string
	Path   string

	PostData map[string]string
	Headers  map[string]string
}

func NewScenario(method, path string) *Scenario {
	return &Scenario{
		Method: method,
		Path:   path,
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

	body, _ := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	_ = body

	w.Success(1)

	return nil
}
