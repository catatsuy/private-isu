package main

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/catatsuy/private-isu/benchmarker/checker"
	"github.com/catatsuy/private-isu/benchmarker/util"
)

func prepareUserdata(userdata string) ([]user, []user, []user, []string, []*checker.Asset, error) {
	if userdata == "" {
		return nil, nil, nil, nil, nil, errors.New("userdataディレクトリが指定されていません")
	}
	info, err := os.Stat(userdata)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if !info.IsDir() {
		return nil, nil, nil, nil, nil, errors.New("userdataがディレクトリではありません")
	}

	file, err := os.Open(userdata + "/names.txt")
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	defer file.Close()

	users := []user{}
	bannedUsers := []user{}

	scanner := bufio.NewScanner(file)
	i := 1
	for scanner.Scan() {
		name := scanner.Text()
		if i%50 == 0 { // 50で割れる場合はbanされたユーザー
			bannedUsers = append(users, user{AccountName: name, Password: name + name})
		} else {
			users = append(users, user{AccountName: name, Password: name + name})
		}
		i++
	}
	adminUsers := users[:9]

	sentenceFile, err := os.Open(userdata + "/kaomoji.txt")
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	defer sentenceFile.Close()

	sentences := []string{}

	sScanner := bufio.NewScanner(sentenceFile)
	for sScanner.Scan() {
		sentence := sScanner.Text()
		sentences = append(sentences, sentence)
	}

	imgs, err := filepath.Glob(userdata + "/img/000*") // 00001.jpg, 00002.png, 00003.gif など
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	images := []*checker.Asset{}

	for _, img := range imgs {
		data, err := ioutil.ReadFile(img)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}

		imgType := ""
		if strings.HasSuffix(img, "jpg") {
			imgType = "image/jpeg"
		} else if strings.HasSuffix(img, "png") {
			imgType = "image/png"
		} else if strings.HasSuffix(img, "gif") {
			imgType = "image/gif"
		}

		images = append(images, &checker.Asset{
			MD5:  util.GetMD5(data),
			Path: img,
			Type: imgType,
		})
	}

	return users[9:], bannedUsers, adminUsers, sentences, images, err
}
