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

func (c Crawler) Crawl(context appengine.Context) {
	// c.Infof("HOSAJFIOJSAOIFJIOSA\n")
	c.ParseDir(c.user_repo, "", context)
}

var mu = &sync.Mutex{}

func (c Crawler) ParseDir(user_repo, path string, context appengine.Context) {
	if len(path) >= len("/test") && path[:len("/test")] == "/test" {
		return
	}
	dirs, files := GetDirInfo(user_repo, path, context)

	// context.Infof("%v\n%v\n", path, dirs)
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
	// context.Infof("close %v\n", path)
}

func (c Crawler) ParseFile(user_repo, file_path string, context appengine.Context) {

	// context.Infof("  %v\n", file_path)
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
	// split := strings.Split(file_path, "/")
	// filename := split[len(split)-1]
	src := string(body)
	c.Analysis.AddFile(file_path, src)

	// context.Infof("  close %v\n", file_path)
	// c.mu.Lock()
	// *c.Files = append(*c.Files, filename)
	// *c.Srcs = append(*c.Srcs, src)
	// c.mu.Unlock()
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

// func logerr(err error) {
// 	if err != nil {
// 		log.Println(err)
// 	}
// }
