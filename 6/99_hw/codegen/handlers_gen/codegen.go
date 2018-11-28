package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
)

type Format struct {
	Url        string `json:"url"`
	Auth       bool   `json:"auth"`
	HTTPMethod string `json:"method"`
}

type Route struct {
	Format
	Func string
}

func parseValidationString(validation string) map[string]string {
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

	fmt.Fprintf(out, `
package `+node.Name.Name+`

import (
	"net/http"
	"fmt"
	"encoding/json"
	"strconv"
	// "io/ioutil"
	// "net/url"
	"context"
)

type Result struct {
`)
	fmt.Fprintln(out, "    Response interface{} `json:\"response,omitempty\"`")
	fmt.Fprintln(out, "    Err string `json:\"error\"`")
	fmt.Fprintln(out, "}")

	fmt.Fprintf(out, `
func WriteResponse (w http.ResponseWriter, res Result, st int) {
	resp, _ := json.Marshal(res)
	w.WriteHeader(st)
	w.Write(resp)
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
				fmt.Fprintln(out, `    err := params.GetParams(r)`)
				fmt.Fprintln(out, `    if err.Err != nil {`)
				fmt.Fprintln(out, `            WriteResponse(w, Result{Err: err.Error()}, err.HTTPStatus)`)
				fmt.Fprintln(out, `            return`)
				fmt.Fprintln(out, `    }`)
				fmt.Fprintln(out, `    err = params.ValidateParams(w, r)`)
				fmt.Fprintln(out, `    if err.Err != nil {`)
				fmt.Fprintln(out, `            WriteResponse(w, Result{Err: err.Error()}, err.HTTPStatus)`)
				fmt.Fprintln(out, `            return`)
				fmt.Fprintln(out, `    }`)

				fmt.Fprintln(out, `    ctx := context.Background()`)
				fmt.Fprintln(out, `    res, e := h.`+name+`(ctx, params)`)
				fmt.Fprintln(out, `    if e != nil {`)
				fmt.Fprintln(out, `        errApi, ok := e.(ApiError)`)
				fmt.Fprintln(out, `        if ok {`)
				fmt.Fprintln(out, `            WriteResponse(w, Result{Err: errApi.Error()}, errApi.HTTPStatus)`)
				fmt.Fprintln(out, `            return`)
				fmt.Fprintln(out, `        }`)
				fmt.Fprintln(out, `    }`)
				fmt.Fprintln(out, `    if e != nil {`)

				fmt.Fprintln(out, `        WriteResponse(w, Result{Err: e.Error()}, 500)`)
				fmt.Fprintln(out, `        return`)
				fmt.Fprintln(out, `    }`)
				fmt.Fprintln(out, `    WriteResponse(w, Result{Response: res}, 200)`)
				fmt.Fprintln(out, `    return`)
				fmt.Fprintln(out, `}`)
				fmt.Fprintln(out)

				rte[strctType] = append(rte[strctType], Route{format, name})
			}

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
				params := ""
				vld := ""

				for _, field := range currStruct.Fields.List {
					if field.Tag == nil {
						continue
					}
					needValidation = (needValidation || field.Tag != nil && strings.HasPrefix(field.Tag.Value, "`apivalidator:\""))
					if needValidation {
						vTag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1]).Get("apivalidator")
						parsed := parseValidationString(vTag)
						name, ok := parsed["paramname"]

						if !ok {
							name = strings.ToLower(field.Names[0].Name)
						}

						required, ok := parsed["required"]
						if ok && required == "required" {
							vld += fmt.Sprintln(`    if p.`+field.Names[0].Name+` == "" {`)
							vld += fmt.Sprintln(`        return ApiError{400, fmt.Errorf("`+name+` must me not empty")}`)
							vld += fmt.Sprintln(`    }`)
						}
						if field.Type.(*ast.Ident).Name == "int" {
							params += fmt.Sprintln(`    name, err := `+`strconv.Atoi(r.FormValue("`+name+`"))`)
							params += fmt.Sprintln(`    if err != nil {`)
							params += fmt.Sprintln(`        return ApiError{400, fmt.Errorf("`+name+` must be int")}`)
							params += fmt.Sprintln(`    }`)
							params += fmt.Sprintln(`    p.`+field.Names[0].Name+` = name`)

							min, ok := parsed["min"]
							if ok {
								vld += fmt.Sprintln(`    if p.`+field.Names[0].Name+` < `+min+` {`)
								vld += fmt.Sprintln(`        return ApiError{400, fmt.Errorf("`+name+` must be`+` >= `+min+`")}`)
								vld += fmt.Sprintln(`    }`)
							}
							max, ok := parsed["max"]
							if ok {
								vld += fmt.Sprintln(`    if p.`+field.Names[0].Name+` > `+max+` {`)
								vld += fmt.Sprintln(`        return ApiError{400, fmt.Errorf("`+name+` must be`+` <= `+max+`")}`)
								vld += fmt.Sprintln(`    }`)
							}
						}

						if field.Type.(*ast.Ident).Name == "string" {
							params += fmt.Sprintln(`    p.`+field.Names[0].Name+` = `+`r.FormValue("`+name+`")`)
							min, ok := parsed["min"]
							if ok {
								vld += fmt.Sprintln(`    if len(p.`+field.Names[0].Name+`) < `+min+` {`)
								vld += fmt.Sprintln(`        return ApiError{400, fmt.Errorf("`+name+` len must be`+` >= `+min+`")}`)
								vld += fmt.Sprintln(`    }`)
							}
							max, ok := parsed["max"]
							if ok {
								vld += fmt.Sprintln(`    if len(p.`+field.Names[0].Name+`) > `+max+` {`)
								vld += fmt.Sprintln(`        return ApiError{400, fmt.Errorf("`+name+` len must be`+` <= `+max+`")}`)
								vld += fmt.Sprintln(`    }`)
							}
						}
						deflt, ok := parsed["default"]
						if ok {
							vld += fmt.Sprintln(`    if p.`+field.Names[0].Name+` == "" {`)
							vld += fmt.Sprintln(` 	       p.`+field.Names[0].Name+` = "`+deflt+`"`)
							vld += fmt.Sprintln(`    }`)
						}
						enum, ok := parsed["enum"]
						if ok {
							enm := strings.Split(enum, "|")
							vld += fmt.Sprintln(`    if `)
							for key, e := range enm {
								vld += fmt.Sprintf(`p.` + field.Names[0].Name + ` != "` + e + `"`)
								if key != len(enm)-1 {
									vld += ` && `
								}
							}
							vld += fmt.Sprintf(`{`)
							vld += fmt.Sprintln(`        return ApiError{400, fmt.Errorf("`+name+` must be one of [`+strings.Join(enm, ", ")+`]")}`)
							vld += fmt.Sprintln(`    }`)
						}
					}
				}
				fmt.Println(params, vld)
				if needValidation {
					fmt.Fprintln(out, `func (p *`+currType.Name.Name+`) GetParams(r *http.Request) ApiError {`)
					fmt.Fprintln(out, params)
					fmt.Fprintln(out, `    return ApiError{Err: nil}`)
					fmt.Fprintln(out, `}`)

					fmt.Fprintln(out, `func (p *`+currType.Name.Name+`) ValidateParams(w http.ResponseWriter, r *http.Request) ApiError {`)
					fmt.Fprintln(out, vld)					
					fmt.Fprintln(out, `    return ApiError{200, nil}`)
					fmt.Fprintln(out, `}`)
				}
			}
		default:
			continue
		}
	}

	for key, rt := range rte {
		fmt.Println(rt)
		fmt.Fprintln(out, `func (h *`+key+`) ServeHTTP(w http.ResponseWriter, r *http.Request) {`)
		fmt.Fprintln(out, `    switch r.URL.Path {`)
		for _, r := range rt {
			fmt.Fprintln(out, `    case "`+r.Format.Url+`": `)
			if r.Format.HTTPMethod != "" {
				fmt.Fprintln(out, `    if r.Method != "`+r.Format.HTTPMethod+`" {`)
				fmt.Fprintln(out, `        WriteResponse(w, Result{Err: "bad method"}, 406)`)
				fmt.Fprintln(out, `        return`)
				fmt.Fprintln(out, `    }`)
			}
			if r.Format.Auth == true {
				fmt.Fprintln(out, `    if r.Header.Get("X-Auth") != "100500" {`)
				fmt.Fprintln(out, `        WriteResponse(w, Result{Err: "unauthorized"}, 403)`)
				fmt.Fprintln(out, `        return`)
				fmt.Fprintln(out, `    }`)
			}
			fmt.Fprintln(out, `        h.handle`+r.Func+`(w, r)`)
		}
		fmt.Fprintln(out, `    default:`)
		fmt.Fprintln(out, `        WriteResponse(w, Result{Err: "unknown method"}, 404)`)
		fmt.Fprintln(out, `    }`)
		fmt.Fprintln(out, `}`)
		fmt.Fprintln(out)
	}
}
