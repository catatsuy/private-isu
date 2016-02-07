package main

import (
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sync/atomic"
)

const (
	UserAgent = "benchmarker"
)

type Worker struct {
	Client    *http.Client
	Transport *http.Transport

	Host string

	Successes int32
	Fails     int32
	Score     int64

	logger *log.Logger
}

func NewWorker(host string) *Worker {
	w := &Worker{
		Host:   host,
		logger: log.New(os.Stdout, "", 0),
	}

	jar, _ := cookiejar.New(&cookiejar.Options{})
	w.Transport = &http.Transport{}
	w.Client = &http.Client{
		Transport: w.Transport,
		Jar:       jar,
	}

	return w
}

func (w *Worker) NewRequest(method, uri string, body io.Reader) (*http.Request, error) {
	parsedURL, err := url.Parse(uri)

	if err != nil {
		return nil, err
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
	}

	if parsedURL.Host == "" {
		parsedURL.Host = w.Host
	}

	req, err := http.NewRequest(method, parsedURL.String(), body)

	if err != nil {
		return nil, err
	}

	return req, err
}

func (w *Worker) SendRequest(req *http.Request, simple bool) (resp *http.Response, err error) {
	reqCh := make(chan bool)

	req.Header.Set("User-Agent", UserAgent)

	go func() {
		resp, err = w.Client.Do(req)
		reqCh <- true
	}()

	<-reqCh

	return resp, err
}

func (w *Worker) Success(point int64) {
	atomic.AddInt32(&w.Successes, 1)
	atomic.AddInt64(&w.Score, point)
}

func (w *Worker) Fail(req *http.Request, err error) error {
	return nil
}
