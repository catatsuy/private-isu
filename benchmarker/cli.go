package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/catatsuy/private-isu/benchmarker/checker"
	"github.com/catatsuy/private-isu/benchmarker/score"
	"github.com/catatsuy/private-isu/benchmarker/util"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota

	FailThreshold     = 5
	InitializeTimeout = time.Duration(10) * time.Second
	BenchmarkTimeout  = 60 * time.Second
	WaitAfterTimeout  = 10 * time.Second

	PostsPerPage = 20
)

// CLI is the command line object
type CLI struct {
	// outStream and errStream are the stdout and stderr
	// to write message from the CLI.
	outStream, errStream io.Writer
}

type user struct {
	AccountName string
	Password    string
}

type Output struct {
	Pass     bool     `json:"pass"`
	Score    int64    `json:"score"`
	Suceess  int64    `json:"success"`
	Fail     int64    `json:"fail"`
	Messages []string `json:"messages"`
}

// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	var (
		target   string
		userdata string

		benchmarkTimeout time.Duration
		waitAfterTimeout time.Duration

		version bool
		debug   bool
	)

	// Define option flag parse
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)

	flags.StringVar(&target, "target", "", "")
	flags.StringVar(&target, "t", "", "(Short)")

	flags.StringVar(&userdata, "userdata", "", "userdata directory")
	flags.StringVar(&userdata, "u", "", "userdata directory")

	flags.DurationVar(&benchmarkTimeout, "benchmark-timeout", BenchmarkTimeout, "benchmark timeout")
	flags.DurationVar(&waitAfterTimeout, "wait-after-timeout", WaitAfterTimeout, "wait after timeout")

	flags.BoolVar(&version, "version", false, "Print version information and quit.")

	flags.BoolVar(&debug, "debug", false, "Debug mode")
	flags.BoolVar(&debug, "d", false, "Debug mode")

	// Parse commandline flag
	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeError
	}

	// Show version
	if version {
		fmt.Fprintf(cli.errStream, "%s version %s\n", Name, Version)
		return ExitCodeOK
	}

	targetHost, err := checker.SetTargetHost(target)
	if err != nil {
		outputNeedToContactUs(err.Error())
		return ExitCodeError
	}

	initialize := make(chan bool)

	setupInitialize(targetHost, initialize)

	users, _, adminUsers, sentences, images, err := prepareUserdata(userdata)
	if err != nil {
		outputNeedToContactUs(err.Error())
		return ExitCodeError
	}

	initReq := <-initialize

	if !initReq {
		fmt.Println(outputResultJSON(false, []string{"初期化リクエストに失敗しました"}))

		return ExitCodeError
	}

	// 最初にDOMチェックなどをやってしまい、通らなければさっさと失敗させる
	commentScenario(checker.NewSession(), randomUser(users), randomUser(users).AccountName, randomSentence(sentences))
	postImageScenario(checker.NewSession(), randomUser(users), randomImage(images), randomSentence(sentences))
	cannotLoginNonexistentUserScenario(checker.NewSession())
	cannotLoginWrongPasswordScenario(checker.NewSession(), randomUser(users))
	cannotAccessAdminScenario(checker.NewSession(), randomUser(users))
	cannotPostWrongCSRFTokenScenario(checker.NewSession(), randomUser(users), randomImage(images))
	loginScenario(checker.NewSession(), randomUser(users))
	banScenario(checker.NewSession(), checker.NewSession(), randomUser(users), randomUser(adminUsers), randomImage(images), randomSentence(sentences))

	if score.GetInstance().GetFails() > 0 {
		fmt.Println(outputResultJSON(false, score.GetFailErrorsStringSlice()))
		return ExitCodeError
	}

	indexMoreAndMoreScenarioCh := makeChanBool(2)
	loadIndexScenarioCh := makeChanBool(2)
	userAndPostPageScenarioCh := makeChanBool(2)
	commentScenarioCh := makeChanBool(1)
	postImageScenarioCh := makeChanBool(1)
	loginScenarioCh := makeChanBool(2)
	banScenarioCh := makeChanBool(1)

	timeoutCh := time.After(benchmarkTimeout)

L:
	for {
		select {
		case <-indexMoreAndMoreScenarioCh:
			go func() {
				indexMoreAndMoreScenario(checker.NewSession())
				indexMoreAndMoreScenarioCh <- true
			}()
		case <-loadIndexScenarioCh:
			go func() {
				loadIndexScenario(checker.NewSession())
				loadIndexScenarioCh <- true
			}()
		case <-userAndPostPageScenarioCh:
			go func() {
				userAndPostPageScenario(checker.NewSession(), randomUser(users).AccountName)
				userAndPostPageScenarioCh <- true
			}()
		case <-commentScenarioCh:
			go func() {
				commentScenario(checker.NewSession(), randomUser(users), randomUser(users).AccountName, randomSentence(sentences))
				commentScenarioCh <- true
			}()
		case <-postImageScenarioCh:
			go func() {
				postImageScenario(checker.NewSession(), randomUser(users), randomImage(images), randomSentence(sentences))
				cannotPostWrongCSRFTokenScenario(checker.NewSession(), randomUser(users), randomImage(images))
				postImageScenarioCh <- true
			}()
		case <-loginScenarioCh:
			go func() {
				loginScenario(checker.NewSession(), randomUser(users))
				cannotLoginNonexistentUserScenario(checker.NewSession())
				cannotLoginWrongPasswordScenario(checker.NewSession(), randomUser(users))
				loginScenarioCh <- true
			}()
		case <-banScenarioCh:
			go func() {
				banScenario(checker.NewSession(), checker.NewSession(), randomUser(users), randomUser(adminUsers), randomImage(images), randomSentence(sentences))
				cannotAccessAdminScenario(checker.NewSession(), randomUser(users))
				banScenarioCh <- true
			}()
		case <-timeoutCh:
			break L
		}
	}

	time.Sleep(waitAfterTimeout)

	var msgs []string
	if !debug {
		msgs = score.GetFailErrorsStringSlice()
	} else {
		msgs = score.GetFailRawErrorsStringSlice()
	}

	fmt.Println(outputResultJSON(true, msgs))

	return ExitCodeOK
}

func outputResultJSON(pass bool, messages []string) string {
	output := Output{
		Pass:     pass,
		Score:    score.GetInstance().GetScore(),
		Suceess:  score.GetInstance().GetSucesses(),
		Fail:     score.GetInstance().GetFails(),
		Messages: messages,
	}

	b, _ := json.Marshal(output)

	return string(b)
}

// 主催者に連絡して欲しいエラー
func outputNeedToContactUs(message string) {
	fmt.Println(outputResultJSON(false, []string{"！！！主催者に連絡してください！！！", message}))
}

func makeChanBool(len int) chan bool {
	ch := make(chan bool, len)
	for i := 0; i < len; i++ {
		ch <- true
	}
	return ch
}

func randomUser(users []user) user {
	return users[util.RandomNumber(len(users))]
}

func randomImage(images []*checker.Asset) *checker.Asset {
	return images[util.RandomNumber(len(images))]
}

func randomSentence(sentences []string) string {
	return sentences[util.RandomNumber(len(sentences))]
}

func setupInitialize(targetHost *url.URL, initialize chan bool) {
	go func(targetHost *url.URL) {
		client := &http.Client{
			Timeout: InitializeTimeout,
		}

		parsedURL := &url.URL{
			Scheme: targetHost.Scheme,
			Host:   targetHost.Host,
			Path:   "/initialize",
		}
		req, err := http.NewRequest("GET", parsedURL.String(), nil)
		if err != nil {
			return
		}

		req.Header.Set("User-Agent", checker.UserAgent)

		res, err := client.Do(req)

		if err != nil {
			initialize <- false
			return
		}
		defer res.Body.Close()
		initialize <- true
	}(targetHost)
}
