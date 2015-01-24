package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

var root = "https://github.com"

func main() {
	c := NewCrawler("vova616", "chipmunk")
	c.Crawl()
	time.Sleep(time.Second * 2)
	c.Analysis.ConstructGraph()

	bts, err := json.Marshal(c.Analysis.Repo)
	logerr(err)
	fmt.Println(string(bts))
	// fmt.Println(c.Files)
	// fmt.Println((*c.Srcs)[0])

	// dirs, files := GetDirInfo(user_repo, "")
	// ParseDir(user_repo, "")
	// for i := 0; i < 20; i++ {
	// time.Sleep(time.Second / 4)
	// _ = os.Stdout.Sync()
	// }
}

type Crawler struct {
	user_repo string
	mu        *sync.Mutex
	// Files     *[]string
	// Srcs      *[]string
	Analysis Analysis
}

func NewCrawler(user, repo string) Crawler {
	c := Crawler{user_repo: "/" + user + "/" + repo}
	c.mu = &sync.Mutex{}
	// c.Files = &[]string{}
	// c.Srcs = &[]string{}
	c.Analysis = NewAnalysis(user, repo)
	return c
}

func (c Crawler) Crawl() {
	c.ParseDir(c.user_repo, "")
}

var mu = &sync.Mutex{}

func (c Crawler) ParseDir(user_repo, path string) {
	dirs, files := GetDirInfo(user_repo, path)
	for _, dir := range dirs {
		go c.ParseDir(user_repo, dir)
	}
	for _, file_path := range files {
		go c.ParseFile(user_repo, file_path)
	}
}

func (c Crawler) ParseFile(user_repo, file_path string) {
	// reset timer (lock mu)
	//fmt.Println(file_path)
	resp, err := http.Get("https://raw.githubusercontent.com/" + user_repo + "/master" + file_path)
	logerr(err)
	body, err := ioutil.ReadAll(resp.Body)
	logerr(err)
	resp.Body.Close()
	split := strings.Split(file_path, "/")
	filename := split[len(split)-1]
	src := string(body)
	c.Analysis.AddFile(filename, src)
	// c.mu.Lock()
	// *c.Files = append(*c.Files, filename)
	// *c.Srcs = append(*c.Srcs, src)
	// c.mu.Unlock()
}

func GetDirInfo(user_repo, path string) (dirs, files []string) {
	resp, err := http.Get(root + user_repo + "/tree/master" + path)
	logerr(err)
	body, err := ioutil.ReadAll(resp.Body)
	logerr(err)
	html := string(body)
	resp.Body.Close()

	dir_re := regexp.MustCompile(`\<a href="` + user_repo + `/tree/master` + path + `([A-Za-z0-9/_]*)" class="js-directory-link"`)
	gofile_re := regexp.MustCompile(`\<a href="` + user_repo + `/blob/master` + path + `([A-Za-z0-9/_]*\.go)" class="js-directory-link"`)
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
