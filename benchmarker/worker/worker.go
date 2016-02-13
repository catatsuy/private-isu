package worker

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/catatsuy/private-isu/benchmarker/score"
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
	Errors    []error

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

func escapeQuotes(s string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(s)
}

func (w *Worker) NewFileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
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

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	// part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	// Content-Typeを指定できないので該当コードから実装
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			escapeQuotes(paramName), escapeQuotes(filepath.Base(path))))
	h.Set("Content-Type", params["type"])
	part, err := writer.CreatePart(h)

	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", parsedURL.String(), body)
	if err == nil {
		req.Header.Add("Content-Type", writer.FormDataContentType())
	} else {
		return nil, err
	}

	return req, err
}

func (w *Worker) RefreshClient() {
	jar, _ := cookiejar.New(&cookiejar.Options{})
	w.Transport = &http.Transport{}
	w.Client = &http.Client{
		Transport: w.Transport,
		Jar:       jar,
	}
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
	score.GetInstance().SetScore(point)
}

func (w *Worker) Fail(req *http.Request, err error) error {
	score.GetInstance().SetFails()
	if req != nil {
		err = fmt.Errorf("%s\tmethod:%s\turi:%s", err, req.Method, req.URL.Path)
	}

	w.Errors = append(w.Errors, err)
	return nil
}
