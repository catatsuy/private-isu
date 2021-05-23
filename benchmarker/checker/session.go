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
	"time"

	"github.com/catatsuy/private-isu/benchmarker/score"
)

const (
	UserAgent = "benchmarker"
)

var (
	targetHost *url.URL
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
		Timeout:   time.Duration(10) * time.Second,
	}

	return w
}

func SetTargetHost(host string) (*url.URL, error) {
	parsedURL, err := urlParse(host)
	if err != nil {
		return nil, err
	}

	targetHost = parsedURL
	return targetHost, nil
}

func urlParse(ref string) (*url.URL, error) {
	u, err := url.Parse(ref)
	if err != nil {
		return nil, err
	}

	if u.Host == "" {
		return nil, fmt.Errorf("host is empty")
	}

	if u.Scheme == "" {
		u.Scheme = "http"
	}

	return &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
	}, nil
}

func (s *Session) NewRequest(method, uri string, body io.Reader) (*http.Request, error) {
	parsedURL, err := url.Parse(uri)

	if err != nil {
		return nil, err
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = targetHost.Scheme
	}

	if parsedURL.Host == "" {
		parsedURL.Host = targetHost.Host
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

func (s *Session) NewFileUploadRequest(uri string, params map[string]string, paramName string, asset *Asset) (*http.Request, error) {
	file, err := os.Open(asset.Path)
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
			escapeQuotes(paramName), escapeQuotes(filepath.Base(asset.Path))))
	h.Set("Content-Type", asset.Type)
	part, err := writer.CreatePart(h)

	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	parsedURL := &url.URL{
		Scheme: targetHost.Scheme,
		Host:   targetHost.Host,
		Path:   uri,
	}

	req, err := http.NewRequest("POST", parsedURL.String(), body)
	if err == nil {
		req.Header.Add("Content-Type", writer.FormDataContentType())
	} else {
		return nil, err
	}

	return req, err
}

func (s *Session) SendRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", UserAgent)

	return s.Client.Do(req)
}

func (s *Session) Success(point int64) {
	score.GetInstance().SetScore(point)
}

func (s *Session) Fail(point int64, req *http.Request, err error) error {
	score.GetInstance().SetFails(point)
	if req != nil {
		err = fmt.Errorf("%s (%s %s)", err, req.Method, req.URL.Path)
	}

	score.GetFailErrorsInstance().Append(err)
	return err
}
