package main

import (
	"bytes"
	"fmt"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

type Scenario struct {
	Method string
	Path   string

	PostData map[string]string
	Headers  map[string]string

	ExpectedStatusCode int
	ExpectedLocation   string
	ExpectedHeaders    map[string]string
	ExpectedAssets     map[string]string
	ExpectedHTML       map[string]string
}

func NewScenario(method, path string) *Scenario {
	return &Scenario{
		Method: method,
		Path:   path,

		ExpectedStatusCode: 200,
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

	doc, _ := goquery.NewDocumentFromReader(res.Body)
	defer res.Body.Close()

	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		url, _ := s.Attr("href")
		fmt.Println(url)
	})

	w.Success(1)

	return nil
}
