package appstract

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"sync"
)

// func main() {
// 	a := NewAnalysis("corgi", "man")
// 	// add async
// 	a.AddFile("file.go", src)
// 	a.AddFile("file2.go", src2)

// 	// after ALL files are added
// 	a.ConstructGraph()

// 	bts, err := json.MarshalIndent(&a.Repo, "", "  ")
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Println(string(bts))

// 	fmt.Println(a.TypesMap)
// 	fmt.Println(a.FilesMap)
// }

// Parse file and get types

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

// type Function struct { // Will be the nodes
// 	FileName, PackageName, Name string
// 	// CallsFrom, CallsTo []*Function
// }

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
			// call_ids = append(call_ids, call_id)
			ls := a.Repo.Pkgs[pkg].Links
			*ls = append(*ls, Link{id, call_id, callerfilename, fname, external})

			return true
		})

		// fmt.Println(id, "---", call_ids)
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

var src string = `
package main

import "fmt"
import "pack"

import ("boo"
        "aa/baa"
        "bat")

type A int

func Add(i, j int) int {
    fmt.Println()
    return Add(i, j)
}

func (a A) Increment() {
    a++
    fmt.Println()
}


func main() {
    a := A(4)
    a.FFF()
    b := B{}
    b.BBB().CCC()
}
`

var src2 string = `
package main

type B struct {}
type C struct {}

func (b B) BBB() C {
    return C{}
}

func (c C) CCC() int {

}


`

// func AddSrcToRepo(src string) {

// 	fset := token.NewFileSet()

// 	f, err := parser.ParseFile(fset, "", src, 0)
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	// ast.Print(fset, f)

// 	// get type info from func decls
// 	// the type info is used to determin the receiver type in method calls
// 	// f().Method() -> we need to know what type f() is. lookup in typemap["pkg.f"]
// 	// filename := "placeholder.go"
// 	for _, decl := range f.Decls {
// 		// functions
// 		if fun, ok := decl.(*ast.FuncDecl); ok {
// 			id, typ := GetDeclInfo(fun, f.Name.Name)
// 			fmt.Println(id, typ)
// 		}
// 	}
// 	fmt.Println()

// 	// get import packages
// 	imports := make([]string, 0)
// 	for _, decl := range f.Decls {
// 		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok.String() == "import" {
// 			for _, s := range gen.Specs {
// 				path := s.(*ast.ImportSpec).Path.Value
// 				path = path[1 : len(path)-1]
// 				split := strings.Split(path, "/")
// 				imprt := split[len(split)-1]
// 				imports = append(imports, imprt)
// 			}
// 		}
// 	}
// 	// fmt.Println(imports)
// 	// fmt.Println()

// 	// connecting
// 	for _, decl := range f.Decls {
// 		fdecl, ok := decl.(*ast.FuncDecl)
// 		if !ok {
// 			continue
// 		}
// 		pkg := f.Name.Name
// 		id, _ := GetDeclInfo(fdecl, pkg)

// 		call_ids := make([]string, 0)
// 		// inspect func decl for calls
// 		ast.Inspect(fdecl, func(n ast.Node) bool {
// 			call, ok := n.(*ast.CallExpr)
// 			if !ok {
// 				return true
// 			}

// 			call_id := ""
// 			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
// 				split := strings.Split(GetType(sel.X), ".")
// 				external := false
// 				for _, imprt := range imports {
// 					if imprt == split[0] {
// 						external = true
// 					}
// 				}
// 				if external {

// 					call_id = GetType(sel.X) + "." + sel.Sel.Name
// 				} else {
// 					call_id = pkg + "." + GetType(sel.X) + "." + sel.Sel.Name
// 				}

// 			} else if fun, ok := call.Fun.(*ast.Ident); ok {
// 				call_id = pkg + "." + fun.Name
// 			}
// 			call_ids = append(call_ids, call_id)
// 			return true
// 		})

// 		fmt.Println(id, "---", call_ids)
// 	}
// }
