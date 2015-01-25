package appstract

import (
	"appengine"
	"appengine/urlfetch"
	"io/ioutil"
	"net/http"
	"regexp"
	"sync"
	"time"
)

type Crawler struct {
	user_repo string
	mu        *sync.Mutex
	Analysis  Analysis
}

func NewCrawler(user, repo string) Crawler {
	c := Crawler{user_repo: "/" + user + "/" + repo}
	c.mu = &sync.Mutex{}
	c.Analysis = NewAnalysis(user, repo)
	return c
}

func (c Crawler) Crawl(context appengine.Context) {
	c.ParseDir(c.user_repo, "", context)
}

var mu = &sync.Mutex{}

func (c Crawler) ParseDir(user_repo, path string, context appengine.Context) {
	dirs, files := GetDirInfo(user_repo, path, context)
	wg := sync.WaitGroup{}
	for _, dir := range dirs {
		wg.Add(1)
		go func(dir string) {
			c.ParseDir(user_repo, dir, context)
			wg.Done()
		}(dir)
	}

	for _, file_path := range files {
		wg.Add(1)

		go func(file_path string) {
			c.ParseFile(user_repo, file_path, context)
			wg.Done()
		}(file_path)
	}
	wg.Wait()
}

func (c Crawler) ParseFile(user_repo, file_path string, context appengine.Context) {
	t := &urlfetch.Transport{Context: context, Deadline: time.Second * 5, AllowInvalidServerCertificate: true}
	client := &http.Client{Transport: t}
	resp, err := client.Get("https://raw.githubusercontent.com/" + user_repo + "/master" + file_path)
	if err != nil || resp == nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	resp.Body.Close()
	src := string(body)
	c.Analysis.AddFile(file_path, src)
}

func GetDirInfo(user_repo, path string, c appengine.Context) (dirs, files []string) {
	// c := appengine.NewContext(r)
	// client := urlfetch.Client(c)
	t := &urlfetch.Transport{Context: c, Deadline: time.Second * 5, AllowInvalidServerCertificate: true}
	client := &http.Client{Transport: t}
	resp, err := client.Get("https://github.com" + user_repo + "/tree/master" + path)
	if err != nil || resp == nil {
		return nil, nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil
	}
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
