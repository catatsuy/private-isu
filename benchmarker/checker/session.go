package checker

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

var (
	targetHost string
)

type Session struct {
	Client    *http.Client
	Transport *http.Transport

	logger *log.Logger
}

func NewSession() *Session {
	w := &Session{
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

func SetTargetHost(host string) error {
	parsedURL, err := url.Parse(host)

	if err != nil {
		return err
	}

	targetHost = ""

	// 完璧にチェックするのは難しい
	if parsedURL.Scheme == "http" {
		targetHost += parsedURL.Host
	} else if parsedURL.Scheme != "" && parsedURL.Scheme != "https" {
		targetHost += parsedURL.Scheme + ":" + parsedURL.Opaque
	} else {
		return fmt.Errorf("不正なホスト名です")
	}

	return nil
}

func (s *Session) NewRequest(method, uri string, body io.Reader) (*http.Request, error) {
	parsedURL, err := url.Parse(uri)

	if err != nil {
		return nil, err
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
	}

	if parsedURL.Host == "" {
		parsedURL.Host = targetHost
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

func (s *Session) NewFileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	parsedURL, err := url.Parse(uri)

	if err != nil {
		return nil, err
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
	}

	if parsedURL.Host == "" {
		parsedURL.Host = targetHost
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

func (s *Session) RefreshClient() {
	jar, _ := cookiejar.New(&cookiejar.Options{})
	s.Transport = &http.Transport{}
	s.Client = &http.Client{
		Transport: s.Transport,
		Jar:       jar,
	}
}

func (s *Session) SendRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", UserAgent)

	return s.Client.Do(req)
}

func (s *Session) Success(point int64) {
	score.GetInstance().SetScore(point)
}

func (s *Session) Fail(req *http.Request, err error) error {
	score.GetInstance().SetFails()
	if req != nil {
		err = fmt.Errorf("%s\tmethod:%s\turi:%s", err, req.Method, req.URL.Path)
	}

	score.GetFailErrorsInstance().Append(err)
	return nil
}
