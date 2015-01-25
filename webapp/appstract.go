package appstract

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	// "time"

	"appengine"
	"appengine/datastore"
	// "sync"
)

type DBPackage struct {
	User, Repo, Path, Name string
	Links                  []Link
}

func extract(user, repo string, r *http.Request) error {
	c := appengine.NewContext(r)
	cr := NewCrawler(user, repo)

	cr.Crawl(r)

	// time.Sleep(time.Second * 180)

	cr.Analysis.ConstructGraph()
	for _, pkg := range cr.Analysis.Repo.Pkgs {
		p := DBPackage{pkg.User, pkg.Repo, pkg.Path, pkg.Name, *pkg.Links}
		key := datastore.NewKey(c, "package", p.User+"/"+p.Repo+"/"+p.Path+p.Name, 0, nil)
		_, err := datastore.Put(c, key, &p)
		if err != nil {
			return err
		}
	}
	return nil
	// bts, err := json.Marshal(c.Analysis.Repo)
	// logerr(err)
	// fmt.Println(string(bts))
}

func root(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	if err := rootTemplate.Execute(w, struct{}{}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	c.Infof("root*** Requested URL: %v", r.URL)
}

func view(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	// q := datastore.NewQuery("Greeting").Ancestor(guestbookKey(c)).Order("-Date").Limit(10)
	// greetings := make([]Greeting, 0, 10)
	// if _, err := q.GetAll(c, &greetings); err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	s := r.FormValue("q")
	if i := strings.Index(s, "github.com/"); i != -1 {
		s = s[i+len("github.com/"):]
	}
	split := strings.Split(s, "/")
	c.Infof("%v\n%v\n", split[0], split[1])
	err := extract(split[0], split[1], r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	c.Infof("")
	if err := viewTemplate.Execute(w, struct{}{}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func analyze(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	p := DBPackage{}
	key := datastore.NewKey(c, "package", "vova616/chipmunk/chipmunk", 0, nil)
	err := datastore.Get(c, key, &p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if err := viewTemplate.Execute(w, p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// http.Redirect(w, r, "/", http.StatusFound)
}

var rootTemplate *template.Template
var viewTemplate *template.Template
var analyzeTemplate *template.Template

func init() {
	var f *os.File
	var s []byte
	var err error
	if f, err = os.Open("static/root.html"); err != nil {
		panic(err)
	}
	if s, err = ioutil.ReadAll(f); err != nil {
		panic(err)
	}
	rootTemplate = template.Must(template.New("root").Parse(string(s)))

	if f, err = os.Open("static/view.html"); err != nil {
		panic(err)
	}
	if s, err = ioutil.ReadAll(f); err != nil {
		panic(err)
	}
	viewTemplate = template.Must(template.New("view").Parse(string(s)))

	if f, err = os.Open("static/analyze.html"); err != nil {
		panic(err)
	}
	if s, err = ioutil.ReadAll(f); err != nil {
		panic(err)
	}
	analyzeTemplate = template.Must(template.New("analyze").Parse(string(s)))

	http.HandleFunc("/analyze", analyze)
	http.HandleFunc("/view", view)
	http.HandleFunc("/", root)
}
