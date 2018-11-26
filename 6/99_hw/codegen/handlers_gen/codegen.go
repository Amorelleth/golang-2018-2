package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strings"
	"text/template"
)

type APIStruct struct {
	Methods map[string]APIMethod
	Name    string
}

type APIMethod struct {
	Name       string
	StructName string
	varType string
	ResultType string
	Format
}

type Vars struct {
	Fields []VarField
	Name   string
}

type VarField struct {
	Vld  string
	Name string
	Type string
	Json string
}

type Format struct {
	Url    string `json:"url"`
	Auth   bool   `json:"auth"`
	Method string `json:"method"`
}

var tmpls []byte
var (
	MethodHandlerTpl = template.Must(template.New("methodTpl").Parse(`	}

	ctx := r.Context()
	res, err := h.{{.Name}}(ctx, params)

	if err != nil {
		fmt.Println("ERROR				", err)
		result, _ := json.Marshal(Result{err.Error(), {{.ResultType}}{}})
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(result)
		return
	}

	result, _ := json.Marshal(Result{"", *res})

	w.WriteHeader(http.StatusOK)
	w.Write(result)

	return
}
`))

	ServeHTTPTpl = template.Must(template.New("ServeHTTPTpl").Parse(
		`func (h *{{.Name}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	{{range .Methods}} case "{{.Format.Url}}":
		h.handle{{.Name}}(w, r)
	{{end}} default:
		w.WriteHeader(404)
		return
    }

	return
}
`))

	VarsTmpl = template.Must(template.New("VarsTmpl").Parse(`

	`))
)

func main() {

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])

	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out) // empty line
	fmt.Fprintln(out, `import "net/http"`)
	fmt.Fprintln(out, `import "fmt"`)
	fmt.Fprintln(out, `import "encoding/json"`)
	fmt.Fprintln(out, `import "strconv"`)
	fmt.Fprintln(out, `import "io/ioutil"`)
	fmt.Fprintln(out, `import "net/url"`)
	fmt.Fprintln(out) // empty line

	strcts := make(map[string]APIStruct, 10)
	smplVars := make(map[string]Vars, 10)
	var format Format

DECLS_LOOP:
	for _, fn := range node.Decls {
		g, ok := fn.(*ast.FuncDecl)
		if ok {
			name := g.Name.Name

			if g.Doc == nil {
				fmt.Printf("SKIP %#v doesnt have comments\n", name)
				continue DECLS_LOOP
			}

			needCodegen := false

			for _, comment := range g.Doc.List {
				comm := comment.Text
				needCodegen = needCodegen || strings.HasPrefix(comm, "// apigen:api")

				if !needCodegen {
					fmt.Printf("SKIP function %#v doesnt have apigen mark\n", name)
					continue DECLS_LOOP
				}

				c := strings.Replace(comm, "// apigen:api", "", 1)
				err := json.Unmarshal([]byte(c), &format)

				if err != nil {
					fmt.Printf("SKIP %#v has invalid json\n", name)
					continue DECLS_LOOP
				}

				if g.Recv == nil {
					fmt.Printf("SKIP function %#v is not a method\n", name)
					continue DECLS_LOOP
				}

				varType := g.Type.Params.List[1].Type.(*ast.Ident).String()
				typ := g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
				resType := g.Type.Results.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name

				_, ok := strcts[typ]

				if ok {
					strcts[typ].Methods[name] = APIMethod{name, typ, varType, resType, format}
				} else {
					strcts[typ] = APIStruct{
						Name: typ,
						Methods: map[string]APIMethod{
							name: APIMethod{name, typ, varType, resType, format},
						},
					}
				}
			}
		}

		// *`required` - поле не должно быть пустым (не должно иметь значение по-умолчанию)
		// * `paramname` - если указано - то брать из параметра с этим именем, иначе `lowercase` от имени
		// * `enum` - "одно из"
		// * `default` - если указано и приходит пустое значение (значение по-умолчанию) - устанавливать то что написано указано в `default`
		// * `min` - >= X для типа `int`, для строк `len(str)` >=
		// * `max` - <= X для типа `int`

		vars, ok := fn.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range vars.Specs {
			currType, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			currStruct, ok := currType.Type.(*ast.StructType)
			if !ok {
				fmt.Printf("SKIP %T is not ast.StructType\n", currStruct)
				continue
			}

			flds := make([]VarField, 0, len(currStruct.Fields.List))
			// needVld := false

			for _, field := range currStruct.Fields.List {
				if field.Tag != nil {
					tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
					vldTag := tag.Get("apivalidator")
					jsonTag := tag.Get("json")

					fmt.Println()
					fmt.Println()
					fmt.Println("json", jsonTag)
					fmt.Println()
					fmt.Println()
					
					if vldTag == "" {
						continue
					}
					// needVld = true

					flds = append(flds, VarField{
						vldTag,
						field.Names[0].String(),
						field.Type.(*ast.Ident).String(),
						jsonTag,
					})
				}
			}

			// if needVld {
			smplVars[currType.Name.String()] = Vars{flds, currType.Name.String()}
			// }
		}
	}

	// fmt.Println(strcts)
	// fmt.Println(Vars)

	for _, val := range strcts {
		// fmt.Fprintln(out, "func (h *" + key + ") ServeHTTP(w http.ResponseWriter, r *http.Request) {")
		
		ServeHTTPTpl.Execute(out, val)

		for _, m := range val.Methods {
			fmt.Fprintln(out)

			fmt.Println(m.Format.Method)
			fmt.Fprintln(out, `// `+m.Name+` handler`)
			fmt.Fprintln(out, `func (h *`+m.StructName+`) handle`+m.Name+` (w http.ResponseWriter, r *http.Request) {`)

			fmt.Fprintln(out, `    type Result struct {`)
			fmt.Fprintln(out, "        Error string `json:\"error\"`")
			fmt.Fprintln(out, "        Response "+m.ResultType+" `json:\"response,omitempty\"`")
			fmt.Fprintln(out, "    }")

			fmt.Fprintf(out, `
	var q url.Values
	if r.Method == "POST" { 
		body, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		query, err := url.ParseQuery(string(body))
		q = query
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
    } else {
		q = r.URL.Query()
	}
`)
			for _, f := range smplVars[m.varType].Fields {
				var fQuery string

				if f.Json == "" {
					fQuery = f.Name
				} else {
					fQuery = f.Json
				}

				fmt.Println()
				fmt.Println()
				fmt.Println(f)
				fmt.Println()
				fmt.Println()


				if f.Type == "int" {
					fmt.Fprintln(out, `    ` + f.Name+`, _ := strconv.Atoi(q.Get("`+ fQuery +`"))`)
					continue
				}
				fmt.Fprintln(out, `    ` + f.Name+` := q.Get("` + fQuery +`")`)
			}
			fmt.Fprintln(out, `    params := `+m.varType+`{`)

			for _, f := range smplVars[m.varType].Fields {
				fmt.Fprintln(out, `        `+f.Name+`: `+f.Name+`,`)
			}

			MethodHandlerTpl.Execute(out, m)
		}
	}
}
