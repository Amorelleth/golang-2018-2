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
				params := []string{}
				vld := []string{}

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
							vld = append(vld, fmt.Sprintf(`    if p.`+field.Names[0].Name+` == "" {`))
							vld = append(vld, fmt.Sprintf(`        return ApiError{400, fmt.Errorf("`+name+` must me not empty")}`))
							vld = append(vld, fmt.Sprintf(`    }`))
						}
						if field.Type.(*ast.Ident).Name == "int" {
							params = append(params, fmt.Sprintf(`    name, err := `+`strconv.Atoi(r.FormValue("`+name+`"))`))
							params = append(params, fmt.Sprintf(`    if err != nil {`))
							params = append(params, fmt.Sprintf(`        return ApiError{400, fmt.Errorf("`+name+` must be int")}`))
							params = append(params, fmt.Sprintf(`    }`))
							params = append(params, fmt.Sprintf(`    p.`+field.Names[0].Name+` = name`))

							min, ok := parsed["min"]
							if ok {
								vld = append(vld, fmt.Sprintf(`    if p.`+field.Names[0].Name+` < `+min+` {`))
								vld = append(vld, fmt.Sprintf(`        return ApiError{400, fmt.Errorf("`+name+` must be`+` >= `+min+`")}`))
								vld = append(vld, fmt.Sprintf(`    }`))
							}
							max, ok := parsed["max"]
							if ok {
								vld = append(vld, fmt.Sprintf(`    if p.`+field.Names[0].Name+` > `+max+` {`))
								vld = append(vld, fmt.Sprintf(`        return ApiError{400, fmt.Errorf("`+name+` must be`+` <= `+max+`")}`))
								vld = append(vld, fmt.Sprintf(`    }`))
							}
						}

						if field.Type.(*ast.Ident).Name == "string" {
							params = append(params, fmt.Sprintf(`    p.`+field.Names[0].Name+` = `+`r.FormValue("`+name+`")`))
							min, ok := parsed["min"]
							if ok {
								vld = append(vld, fmt.Sprintf(`    if len(p.`+field.Names[0].Name+`) < `+min+` {`))
								vld = append(vld, fmt.Sprintf(`        return ApiError{400, fmt.Errorf("`+name+` len must be`+` >= `+min+`")}`))
								vld = append(vld, fmt.Sprintf(`    }`))
							}
							max, ok := parsed["max"]
							if ok {
								vld = append(vld, fmt.Sprintf(`    if len(p.`+field.Names[0].Name+`) > `+max+` {`))
								vld = append(vld, fmt.Sprintf(`        return ApiError{400, fmt.Errorf("`+name+` len must be`+` <= `+max+`")}`))
								vld = append(vld, fmt.Sprintf(`    }`))
							}
						}
						deflt, ok := parsed["default"]
						if ok {
							vld = append(vld, fmt.Sprintf(`    if p.`+field.Names[0].Name+` == "" {`))
							vld = append(vld, fmt.Sprintf(` 	       p.`+field.Names[0].Name+` = "`+deflt+`"`))
							vld = append(vld, fmt.Sprintf(`    }`))
						}
						enum, ok := parsed["enum"]
						if ok {
							enm := strings.Split(enum, "|")
							init := fmt.Sprintf(`    if `)
							for key, e := range enm {
								init += `p.` + field.Names[0].Name + ` != "` + e + `"`
								if key != len(enm)-1 {
									init += ` && `
								}
							}
							init += fmt.Sprintf(` {`)
							vld = append(vld, init)
							vld = append(vld, fmt.Sprintf(`        return ApiError{400, fmt.Errorf("`+name+` must be one of [`+strings.Join(enm, ", ")+`]")}`))
							vld = append(vld, fmt.Sprintf(`    }`))
						}
					}
				}
				fmt.Println(params, vld)
				if needValidation {
					fmt.Fprintln(out, `func (p *`+currType.Name.Name+`) GetParams(r *http.Request) ApiError {`)
					for _, p := range params {
						fmt.Fprintln(out, p)
					}
					fmt.Fprintln(out, `    return ApiError{Err: nil}`)
					fmt.Fprintln(out, `}`)

					fmt.Fprintln(out, `func (p *`+currType.Name.Name+`) ValidateParams(w http.ResponseWriter, r *http.Request) ApiError {`)
					for _, v := range vld {
						fmt.Fprintln(out, v)
					}
					fmt.Fprintln(out, `    return ApiError{200, nil}`)
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
