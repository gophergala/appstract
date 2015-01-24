package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
)

var root = "https://github.com"

func main() {
	user_repos := "/golang/go"

	// dirs, files := GetDirInfo(user_repos, "")
	ParseDir(user_repos, "")
	for i := 0; i < 20; i++ {
		time.Sleep(time.Second / 4)
		_ = os.Stdout.Sync()
	}
}

var mu = &sync.Mutex{}

func ParseDir(user_repos, path string) {
	dirs, files := GetDirInfo(user_repos, path)
	for _, dir := range dirs {
		go ParseDir(user_repos, dir)
	}
	for _, file_path := range files {
		go ParseFile(user_repos, file_path)
	}
}

func ParseFile(user_repos, file_path string) {
	fmt.Println(file_path)
	// resp, err := http.Get("https://raw.githubusercontent.com/" + user_repos + "/master" + file_path)
	// logerr(err)
	// body, err := ioutil.ReadAll(resp.Body)
	// logerr(err)
	// resp.Body.Close()

	//	html := string(body)

	// sent to analyzer (grapher.go)
}

func GetDirInfo(user_repos, path string) (dirs, files []string) {
	resp, err := http.Get(root + user_repos + "/tree/master" + path)
	logerr(err)
	body, err := ioutil.ReadAll(resp.Body)
	logerr(err)
	html := string(body)
	resp.Body.Close()

	dir_re := regexp.MustCompile(`\<a href="` + user_repos + `/tree/master` + path + `([A-Za-z0-9/_]*)" class="js-directory-link"`)
	gofile_re := regexp.MustCompile(`\<a href="` + user_repos + `/blob/master` + path + `([A-Za-z0-9/_]*\.go)" class="js-directory-link"`)
	for _, match := range dir_re.FindAllStringSubmatch(html, -1) {
		dirs = append(dirs, path+match[1])
	}
	for _, match := range gofile_re.FindAllStringSubmatch(html, -1) {
		files = append(files, path+match[1])
	}

	return dirs, files
}

func logerr(err error) {
	if err != nil {
		log.Println(err)
	}
}
