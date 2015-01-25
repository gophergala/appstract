package appstract

import (
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	// "time"

	"appengine"
	"appengine/datastore"
	// "sync"
)

type TemplateData struct {
	Packages []PackageTemplate
	Package  DBPackage
}

type PackageTemplate struct {
	Path string
	Name string
}

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
		if len(*pkg.Links) == 0 {
			continue
		}
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
	s := r.FormValue("q")
	if s == "" {
		s = r.URL.Path[len("/view/"):]
		if s != "" && s[len(s)-1] == '/' {
			s = s[:len(s)-1]
		}
	}
	if i := strings.Index(s, "github.com/"); i != -1 {
		s = s[i+len("github.com/"):]
	}
	split := strings.Split(s, "/")

	if len(split) < 2 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	c := appengine.NewContext(r)
	c.Infof("\n\n\n%s\n\n\n\n", s)
	p := DBPackage{}
	if len(split) > 2 {
		key := datastore.NewKey(c, "package", s, 0, nil)
		if err := datastore.Get(c, key, &p); err != nil {
			split = split[:2]
		}
	}
	if len(split) == 2 {
		k1 := datastore.NewKey(c, "package", s+"/", 0, nil)
		k2 := datastore.NewKey(c, "package", s+"/zzzzzzzzzzzzz", 0, nil)
		q := datastore.NewQuery("package").Filter("__key__ >", k1).Filter("__key__ <", k2).Limit(1)

		ps := make([]DBPackage, 0, 1)
		if _, err := q.GetAll(c, &ps); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(ps) == 0 {
			serve404(w)
			return
		}
		p = ps[0]
	}

	strs, err := GetPackages(split[0]+"/"+split[1], r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	pkgs := make([]PackageTemplate, 0, len(strs))
	for _, s := range strs {
		path := s
		split := strings.Split(s, "/")
		s = strings.Join(split[2:], "/")
		pkgs = append(pkgs, PackageTemplate{path, s})
	}

	templateData := TemplateData{pkgs, p}
	if err := viewTemplate.Execute(w, templateData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func analyze(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	s := r.FormValue("q")
	if i := strings.Index(s, "github.com/"); i != -1 {
		s = s[i+len("github.com/"):]
	}
	split := strings.Split(s, "/")

	if err := extract(split[0], split[1], r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	c.Infof("")
	if err := viewTemplate.Execute(w, struct{}{}); err != nil {
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

	http.HandleFunc("/", root)
	http.HandleFunc("/view/", view)
	http.HandleFunc("/analyze/", analyze)
}

func GetPackages(s string, r *http.Request) ([]string, error) {
	c := appengine.NewContext(r)
	k1 := datastore.NewKey(c, "package", s+"/", 0, nil)
	k2 := datastore.NewKey(c, "package", s+"/zzzzzzzzzzzzz", 0, nil)
	q := datastore.NewQuery("package").Filter("__key__ >", k1).Filter("__key__ <", k2).KeysOnly()

	keys, err := q.GetAll(c, nil)
	if err != nil {
		return nil, err
	}
	strs := make([]string, 0, len(keys))
	for _, k := range keys {
		s := k.StringID()
		strs = append(strs, s)
	}

	c.Infof("\n\n\n%v\n\n\n", s)
	c.Infof("\n\n\n%v\n\n\n", strs)
	return strs, nil
}

func serve404(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, "Not Found")
}
