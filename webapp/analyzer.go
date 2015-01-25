package appstract

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"sync"
)

type Analysis struct {
	mu       *sync.Mutex
	Files    map[string]*ast.File
	TypesMap map[string]string
	FilesMap map[string]string
	Repo     Repository
}

type Repository struct {
	User, Name string
	Pkgs       map[string]Package
}

type Package struct {
	User, Repo, Path, Name string
	// Fns        map[string]Function
	Links *[]Link
}

type Link struct {
	Source    string `json:"s"`
	Target    string `json:"t"`
	SFileName string `json:"sf"`
	TFileName string `json:"tf"`
	External  bool   `json:"ex"`
}

func NewAnalysis(user, repo string) Analysis {
	a := Analysis{}
	a.mu = &sync.Mutex{}
	a.Files = make(map[string]*ast.File)
	a.TypesMap = make(map[string]string)
	a.FilesMap = make(map[string]string)
	a.Repo = Repository{user, repo, make(map[string]Package)}
	return a
}

func (a Analysis) AddFile(filename, src string) {
	f := ParseFile(filename, src)
	if f == nil || f.Name == nil {
		return
	}
	a.mu.Lock()
	a.Files[filename] = f
	pkgname := f.Name.Name
	for _, decl := range f.Decls {
		// functions
		if fun, ok := decl.(*ast.FuncDecl); ok {
			id, typ := GetDeclInfo(fun, pkgname)
			a.TypesMap[pkgname+"."+id] = typ
			a.FilesMap[pkgname+"."+id] = filename
		}
	}
	a.mu.Unlock()
}

func (a Analysis) ConstructGraph() {
	for fname, file := range a.Files {
		a.AddToGraph(fname, file)
	}
}

func (a Analysis) AddToGraph(callerfilename string, f *ast.File) {
	pkg := f.Name.Name
	split := strings.Split(pkg, "/")
	pkgpath := strings.Join(split[:len(split)-1], "/")
	if _, ok := a.Repo.Pkgs[pkg]; !ok {
		a.Repo.Pkgs[pkg] = Package{a.Repo.User, a.Repo.Name, pkgpath, pkg, &[]Link{}}
	}
	imports := make([]string, 0)
	for _, decl := range f.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok.String() == "import" {
			for _, s := range gen.Specs {
				path := s.(*ast.ImportSpec).Path.Value
				path = path[1 : len(path)-1]
				split := strings.Split(path, "/")
				imprt := split[len(split)-1]
				imports = append(imports, imprt)
			}
		}
	}

	for _, decl := range f.Decls {
		fdecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		id, _ := GetDeclInfo(fdecl, pkg)

		// call_ids := make([]string, 0)
		// inspect func decl for calls
		ast.Inspect(fdecl, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			fname := ""
			call_id := ""
			external := false
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				// call can be external or in package
				split := strings.Split(GetType(sel.X), ".")
				for _, imprt := range imports {
					if imprt == split[0] {
						external = true
					}
				}

				typ := ""
				if external {
					typ = GetType(sel.X)
					if t, ok := a.TypesMap[typ]; ok {
						typ = split[0] + "." + t
					}
					fname = split[0]
				} else {
					typ = pkg + "." + GetType(sel.X)

					if t, ok := a.TypesMap[typ]; ok {
						typ = t

					} else {
						typ = GetType(sel.X)
						fname = callerfilename
					}
					if fn, ok := a.FilesMap[pkg+"."+typ+"."+sel.Sel.Name]; ok {
						fname = fn
					} else {
						fname = callerfilename
					}

				}

				call_id = typ + "." + sel.Sel.Name
			} else if fun, ok := call.Fun.(*ast.Ident); ok {
				call_id = fun.Name
				fname = callerfilename

				for _, c := range []string{"len", "append", "make", "cap", "panic", "int", "string", "float64", "float32", "byte"} {
					if c == call_id {
						return true
					}
				}

			}
			ls := a.Repo.Pkgs[pkg].Links
			*ls = append(*ls, Link{id, call_id, callerfilename, fname, external})

			return true
		})
	}
}

func ParseFile(file, src string) *ast.File {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, file, src, 0)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	// ast.Print(fset, f)
	return f
}

func GetDeclInfo(fun *ast.FuncDecl, pkg string) (id, typ string) {
	rec := ""
	if fun.Recv != nil {
		if star, ok := fun.Recv.List[0].Type.(*ast.StarExpr); ok {
			rec = star.X.(*ast.Ident).Name
		}
		if ident, ok := fun.Recv.List[0].Type.(*ast.Ident); ok {
			rec = ident.Name
		}
	}

	// id = pkg + "."
	if rec != "" {
		id += rec + "."
	}
	id += fun.Name.Name

	typ = ""
	if fun.Type.Results != nil {
		if ident, ok := fun.Type.Results.List[0].Type.(*ast.Ident); ok {
			typ = ident.Name
		}
	}

	return id, typ
}

func GetType(n interface{}) (s string) {
	switch e := n.(type) {
	case *ast.IndexExpr:
		s = GetType(e.X)
	case *ast.ParenExpr:
		s = GetType(e.X)
	case *ast.SliceExpr:
		s = GetType(e.X)
	case *ast.StarExpr:
		//return "*" + GetType(e.X)
		// don't care for pointers for this application
		s = GetType(e.X)
	case *ast.UnaryExpr:
		s = GetType(e.X)
	case *ast.BinaryExpr:
		s = GetType(e.X)
	case *ast.Ident:
		if e.Obj != nil {
			s = GetType(e.Obj.Decl)
		} else {
			s = e.Name
		}
	case *ast.Field:
		s = GetType(e.Type)
	case *ast.FuncDecl:
		s = GetType(e.Type.Results.List[0].Type)
	case *ast.TypeSpec:
		s = e.Name.Name
	case *ast.ValueSpec:
		s = GetType(e.Type)
	case *ast.CallExpr:
		// if selex, ok := e.Fun(*ast.SelectorExpr); ok {
		// if selex.X is a package and function "pkg.funname" is not found in type map {
		// return e.Fun.X + "." + e.Sel.Name + "()"
		// }
		s = GetType(e.Fun)
	case *ast.SelectorExpr:
		s = GetType(e.X) + "." + GetType(e.Sel)
		// case *ast.TypeAssertExpr:
		//  return GetType(e.X)
	case *ast.AssignStmt:
		s = GetType(e.Rhs[0])
	case *ast.ArrayType:
		s = GetType(e.Elt)
	case *ast.CompositeLit:
		s = GetType(e.Type)
	default:
		fmt.Printf("UnimplementedType: %T\n", n)
		return "UnimplementedTypeDetection"
	}
	return s
}
