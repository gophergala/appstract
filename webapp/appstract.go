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

func extract(user, repo string, c appengine.Context) bool {
	// c.Infof("%s---%s---HOSAJFIOJSAOIFJIOSA\n", user, repo)
	cr := NewCrawler(user, repo)
	// c.Infof("%s---%s---HOSAJFIOJSAOIFJIOSA\n", user, repo)
	cr.Crawl(c)

	// time.Sleep(time.Second * 20)

	cr.Analysis.ConstructGraph()

	for _, pkg := range cr.Analysis.Repo.Pkgs {
		if len(*pkg.Links) == 0 {
			continue
		}
		p := DBPackage{pkg.User, pkg.Repo, pkg.Path, pkg.Name, *pkg.Links}
		key := datastore.NewKey(c, "package", p.User+"/"+p.Repo+"/"+p.Path+p.Name, 0, nil)
		_, _ = datastore.Put(c, key, &p)

	}
	if len(cr.Analysis.Repo.Pkgs) == 0 {
		return false
	}
	return true
	// bts, err := json.Marshal(c.Analysis.Repo)
	// logerr(err)
	// fmt.Println(string(bts))
}

func root(w http.ResponseWriter, r *http.Request) {
	// c := appengine.NewContext(r)
	if err := rootTemplate.Execute(w, struct{}{}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
	// c.Infof("\n\n\n%s\n\n\n\n", s)
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
			// serve404(w)
			user_repo := split[0] + "/" + split[1]
			if ok := extract(split[0], split[1], c); !ok {
				serve404(w, user_repo)
				return
			}
			http.Redirect(w, r, "/view/"+user_repo, http.StatusFound)
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
	// c := appengine.NewContext(r)

	s := r.URL.Path[len("/analyze/"):]
	if s != "" && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}

	split := strings.Split(s, "/")
	if len(split) < 2 {
		http.Redirect(w, r, "/", http.StatusOK)
	}

	if err := analyzeTemplate.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func waiting(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Path[len("/waiting/"):]
	if s != "" && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}

	split := strings.Split(s, "/")
	if len(split) < 2 {
		http.Redirect(w, r, "/", http.StatusOK)
	}
	// user_repo := split[0] + "/" + split[1]
	// for i := 0; i < 300; i++ {
	// 	if !processing[user_repo] {
	// 		http.Redirect(w, r, "/view/"+user_repo, http.StatusFound)
	// 		break
	// 	}
	// 	time.Sleep(time.Second)
	// }

	// if err := viewTemplate.Execute(w, struct{}{}); err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// }
	http.Redirect(w, r, "/", http.StatusFound)
}

var rootTemplate *template.Template
var viewTemplate *template.Template
var analyzeTemplate *template.Template

func init() {

	var f *os.File
	var s []byte
	var err error
	if f, err = os.Open("templates/root.html"); err != nil {
		panic(err)
	}
	if s, err = ioutil.ReadAll(f); err != nil {
		panic(err)
	}
	rootTemplate = template.Must(template.New("root").Parse(string(s)))

	if f, err = os.Open("templates/view.html"); err != nil {
		panic(err)
	}
	if s, err = ioutil.ReadAll(f); err != nil {
		panic(err)
	}
	viewTemplate = template.Must(template.New("view").Parse(string(s)))

	if f, err = os.Open("templates/analyze.html"); err != nil {
		panic(err)
	}
	if s, err = ioutil.ReadAll(f); err != nil {
		panic(err)
	}
	analyzeTemplate = template.Must(template.New("analyze").Parse(string(s)))

	http.HandleFunc("/", root)
	http.HandleFunc("/view/", view)
	http.HandleFunc("/analyze/", analyze)
	http.HandleFunc("/waiting/", waiting)
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

	// c.Infof("\n\n\n%v\n\n\n", s)
	// c.Infof("\n\n\n%v\n\n\n", strs)
	return strs, nil
}

func serve404(w http.ResponseWriter, s string) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, "Invalid GitHub repository: https://github.com/"+s+". Make sure the repository contains go files")
}
