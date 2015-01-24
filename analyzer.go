package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

func main() {
	a := NewAnalysis()
	// add async
	a.AddFile("file.go", src)
	a.AddFile("file2.go", src2)

	// after ALL files are added

	fmt.Println(a.Types)
}

// Parse file and get types

type Analysis struct {
	Files map[string]*ast.File
	Types map[string]string
}

func NewAnalysis() Analysis {
	return Analysis{make(map[string]*ast.File), make(map[string]string)}
}

func (a Analysis) AddFile(filename, src string) {
	f := ParseFile(filename, src)
	a.Files[filename] = f

	pkgname := f.Name.Name
	for _, decl := range f.Decls {
		// functions
		if fun, ok := decl.(*ast.FuncDecl); ok {
			id, typ := GetDeclInfo(fun, pkgname)
			a.Types[id] = typ
		}
	}
}

func ParseFile(file, src string) *ast.File {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, file, src, 0)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return f
}

func AddSrcToRepo(src string) {

	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	// ast.Print(fset, f)

	// get type info from func decls
	// the type info is used to determin the receiver type in method calls
	// f().Method() -> we need to know what type f() is. lookup in typemap["pkg.f"]
	// filename := "placeholder.go"
	for _, decl := range f.Decls {
		// functions
		if fun, ok := decl.(*ast.FuncDecl); ok {
			id, typ := GetDeclInfo(fun, f.Name.Name)
			fmt.Println(id, typ)
		}
	}
	fmt.Println()

	// get import packages
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
	// fmt.Println(imports)
	// fmt.Println()

	// connecting
	for _, decl := range f.Decls {
		fdecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		pkg := f.Name.Name
		id, _ := GetDeclInfo(fdecl, pkg)

		call_ids := make([]string, 0)
		// inspect func decl for calls
		ast.Inspect(fdecl, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			call_id := ""
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				split := strings.Split(GetType(sel.X), ".")
				external := false
				for _, imprt := range imports {
					if imprt == split[0] {
						external = true
					}
				}
				if external {

					call_id = GetType(sel.X) + "." + sel.Sel.Name
				} else {
					call_id = pkg + "." + GetType(sel.X) + "." + sel.Sel.Name
				}

			} else if fun, ok := call.Fun.(*ast.Ident); ok {
				call_id = pkg + "." + fun.Name
			}
			call_ids = append(call_ids, call_id)
			return true
		})

		fmt.Println(id, "---", call_ids)
	}
}

func GetDeclInfo(fun *ast.FuncDecl, pkg string) (id, typ string) {
	rec := ""
	if fun.Recv != nil {
		rec = fun.Recv.List[0].Type.(*ast.Ident).Name
	}

	id = pkg + "."
	if rec != "" {
		id += rec + "."
	}
	id += fun.Name.Name

	typ = ""
	if fun.Type.Results != nil {
		typ = fun.Type.Results.List[0].Type.(*ast.Ident).Name
	}

	return id, typ
}

func GetType(n interface{}) string {
	switch e := n.(type) {
	case *ast.IndexExpr:
		return GetType(e.X)
	case *ast.ParenExpr:
		return GetType(e.X)
	case *ast.SliceExpr:
		return GetType(e.X)
	case *ast.StarExpr:
		//return "*" + GetType(e.X)
		// don't care for pointers for this application
		return GetType(e.X)
	case *ast.UnaryExpr:
		return GetType(e.X)
	case *ast.BinaryExpr:
		return GetType(e.X)
	case *ast.Ident:
		if e.Obj != nil {
			return GetType(e.Obj.Decl)
		} else {
			return e.Name
		}
	case *ast.Field:
		return GetType(e.Type)
	case *ast.FuncDecl:
		return GetType(e.Type.Results.List[0].Type)
	case *ast.TypeSpec:
		return e.Name.Name
	case *ast.ValueSpec:
		return GetType(e.Type)
	case *ast.CallExpr:
		// if selex, ok := e.Fun(*ast.SelectorExpr); ok {
		// if selex.X is a package and function "pkg.funname" is not found in type map {
		// return e.Fun.X + "." + e.Sel.Name + "()"
		// }
		return GetType(e.Fun)
	case *ast.SelectorExpr:
		return GetType(e.X) + "." + GetType(e.Sel)
		// case *ast.TypeAssertExpr:
		//  return GetType(e.X)
	case *ast.AssignStmt:
		return GetType(e.Rhs[0])
	case *ast.ArrayType:
		return GetType(e.Elt)
	default:
		fmt.Printf("UnimplementedType: %T\n", n)
		return "UnimplementedTypeDetection"
	}
	return ""
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
    b := pack.B{}
    b.BBB().CCC()
}
`

var src2 string = `
package pack

type B struct {}
type C struct {}

func (b B) BBB() C {
    return C{}
}

func (c C) CCC() int {

}


`
