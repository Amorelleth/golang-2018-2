package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	// "log"
	"os"
	"reflect"
	"strings"
	// "net/http"
)

type Format struct {
	Url    string `json:"url"`
	Auth   bool   `json:"auth"`
	HTTPMethod string `json:"method"`
}

type Route struct {
	Format
	Func string
}

func parseValidationString(validation string) map[string]string{
	v := strings.Split(validation, ",")
	res := make(map[string]string, len(v))

	for _, val := range v {
		vld := strings.Split(val, "=")

		if len(vld) > 1 {
			res[vld[0]] = vld[1]
		} else {
			res[vld[0]] = vld[0]
		}
	}

	return res
}

func main() {
	fset := token.NewFileSet()
	node, _ := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)

	out, _ := os.Create(os.Args[2])

	fmt.Fprintln(out, `
package `+node.Name.Name+`

import (
	"net/http"
	// "fmt"
	"encoding/json"
	// "strconv"
	// "io/ioutil"
	// "net/url"
	// "context"
)

type Result struct {
	Error string
	Response User
}
`)

	rte := make(map[string][]Route)

	for _, d := range node.Decls {
		switch d.(type) {
		case *ast.FuncDecl:
			fn, _ := d.(*ast.FuncDecl)
			if fn.Doc == nil {
				continue
			}

			name := fn.Name.Name
			needCodegen := false
			for _, comment := range fn.Doc.List {
				com := comment.Text
				needCodegen = (needCodegen || strings.HasPrefix(com, "// apigen:api")) && fn.Recv != nil

				if !needCodegen {
					continue
				}

				strctType := fn.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
				paramsType := fn.Type.Params.List[len(fn.Type.Params.List)-1].Type.(*ast.Ident).Name

				format := Format{}
				err := json.Unmarshal([]byte(strings.Replace(com, "// apigen:api", "", 1)), &format)

				if err != nil {
					continue
				}

				fmt.Fprintln(out, `func (h *`+strctType+`) handle`+name+`(w http.ResponseWriter, r *http.Request) {`)
				fmt.Fprintln(out, `    params := `+paramsType+`{}`)
				fmt.Fprintln(out, `    params.getParams(r *http.Request)`)
				fmt.Fprintln(out, `    params.ValidateParams()`)
				fmt.Fprintln(out, `    res, err := h.`+name+`(ctx, params)`)
				fmt.Fprintln(out, `    if err != nil {`)
				fmt.Fprintln(out, `        w.WriteHeader(400)`)
				fmt.Fprintln(out, `    }`)
				fmt.Fprintln(out, `    w.WriteHeader(200)`)
				fmt.Fprintln(out, `    resp, _ := json.Marshal(res)`)
				fmt.Fprintln(out, `    w.Write(resp)`)
				fmt.Fprintln(out, `}`)
				fmt.Fprintln(out)

				rte[strctType] = append(rte[strctType], Route{format, name})

				// resType := fn.Type.Results.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name

			}
		// *`required` - поле не должно быть пустым (не должно иметь значение по-умолчанию)
		// * `paramname` - если указано - то брать из параметра с этим именем, иначе `lowercase` от имени
		// * `enum` - "одно из"
		// * `default` - если указано и приходит пустое значение (значение по-умолчанию) - устанавливать то что написано указано в `default`
		// * `min` - >= X для типа `int`, для строк `len(str)` >=
		// * `max` - <= X для типа `int`

		case *ast.GenDecl:
			vr, _ := d.(*ast.GenDecl)

			for _, spec := range vr.Specs {
				currType, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				currStruct, ok := currType.Type.(*ast.StructType)

				if !ok {
					continue
				}

				needValidation := false
				params := []string{}

				for _, field := range currStruct.Fields.List {
						if field.Tag == nil {
							continue
						}
						
						tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
						vTag := tag.Get("apivalidator")
						needValidation = tag.Get("apivalidator") != ""

						parsed := parseValidationString(vTag)

						name, ok := parsed["paramname"]
						if !ok {
							name = strings.ToLower(field.Names[0].Name)
						}

						if field.(*ast.StarExpr) == "int"
						params = append(params, fmt.Sprintf(`    h.`+field.Names[0].Name+` = `+`r.FormValue("`+name+`")`))
					}
					if needValidation {
						fmt.Fprintln(out, `func (h *`+currType.Name.Name+`) GetParams(r *http.Request) {`)
						for _, p := range params {
							fmt.Fprintln(out, p)
						}
						fmt.Fprintln(out, `}`)
					}
				}
		default:
			continue
		}
	}

	for key, rt := range rte {
		fmt.Fprintln(out, `func (h *`+key+`) ServeHTTP(w http.ResponseWriter, r *http.Request) {`)
		fmt.Fprintln(out, `    switch r.URL.Path {`)
		for _, r := range rt {
			fmt.Fprintln(out, `    case "`+r.Format.Url+`": `)
			if r.Format.Auth == true {
				fmt.Fprintln(out, `    if r.Header.Get("X-Auth") != "100500" {`)
				fmt.Fprintln(out, `        w.WriteHeader(http.StatusForbidden)`)
				fmt.Fprintln(out, `    }`)
				continue
			}
			fmt.Fprintln(out, `        h.handle`+r.Func+`(w, r)`)
		}
		fmt.Fprintln(out, `    default:`)
		fmt.Fprintln(out, `        w.WriteHeader(404)`)
		fmt.Fprintln(out, `    }`)
		fmt.Fprintln(out, `}`)
		fmt.Fprintln(out)
	}
}
