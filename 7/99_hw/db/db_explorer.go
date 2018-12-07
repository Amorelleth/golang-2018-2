package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type TableInfo struct {
	Field      string
	Type       string
	Collation  interface{}
	Null       string
	Key        string
	Default    interface{}
	Extra      string
	Privileges string
	Comment    string
}

type Field struct {
	Name       string
	Type       string
	IsRequired bool
}

type PrimaryKey struct {
	Name string
	Auto bool
}

type Table struct {
	Fields     []Field
	PrimaryKey PrimaryKey
}

type Handler struct {
	DB            *sql.DB
	Tables        map[string]Table `json:"tables"`
	DefaultLimit  string
	DefaultOffset string
}

type Result struct {
	Resp interface{} `json:"response,omitempty"`
	Err  interface{} `json:"error,omitempty"`
}

func WriteResponse(w http.ResponseWriter, res Result, status int) {
	resp, _ := json.Marshal(res)
	w.WriteHeader(status)
	w.Write(resp)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	ti, _ := regexp.MatchString(`^\/\w+\/\d+($|\/?\??)`, url)
	if ti {
		h.handleTableIdRequest(w, r)
		return
	}

	t, _ := regexp.MatchString(`^\/\w+($|\/?\??)`, url)
	if t {
		h.handleTableRequest(w, r)
		return
	}

	switch url {
	case "/", "/?":
		h.handleTablesList(w, r)
	default:
		WriteResponse(w, Result{Err: "unknown path"}, 404)
		return
	}
}

func (h *Handler) handleTableRequest(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path, "/")
	tn := path[1]
	te := false

	for key, _ := range h.Tables {
		if tn == key {
			te = true
			break
		}
	}

	if !te {
		WriteResponse(w, Result{Err: "unknown table"}, 404)
		return
	}

	table := h.Tables[tn]

	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		limit := q.Get("limit")
		offset := q.Get("offset")

		_, err := strconv.Atoi(limit)
		if err != nil {
			limit = h.DefaultLimit
		}
		_, err = strconv.Atoi(offset)
		if err != nil {
			offset = h.DefaultOffset
		}

		rows, _ := h.DB.Query("select * from `" + tn + "` limit " + limit + " offset " + offset)
		resp := []interface{}{}
		for rows.Next() {
			table := h.Tables[tn]
			rfmt := table.FormatRows()
			rows.Scan(rfmt...)
			resp = append(resp, table.ConvertRows(rfmt))
		}
		rows.Close()

		WriteResponse(w, Result{Resp: map[string]interface{}{"records": resp}}, 200)
		return

	case http.MethodPut:
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		params := make(map[string]interface{}, len(table.Fields))
		decoder.Decode(&params)
		err := table.ValidateParams(params)

		if err != nil {
			WriteResponse(w, Result{Err: err.Error()}, 400)
			return
		}

		valsToInsert := table.FormatValsToInsert(params)
		fields := make([]string, len(table.Fields))
		vals := make([]string, len(table.Fields))

		for i, field := range table.Fields {
			fields[i] = "`" + field.Name + "`"
			vals[i] = "?"
		}
		res, err := h.DB.Exec(
			fmt.Sprintf("insert into `%s` (%s) values (%s)", tn, strings.Join(fields, ", "), strings.Join(vals, ", ")),
			valsToInsert...,
		)
		if err != nil {
			WriteResponse(w, Result{}, 500)
			return
		}

		id, _ := res.LastInsertId()
		WriteResponse(w, Result{Resp: map[string]interface{}{table.PrimaryKey.Name: id}}, 200)
		return
	default:
		WriteResponse(w, Result{}, 400)
		return
	}
}

func (h *Handler) handleTableIdRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	p := strings.Split(path, "/")
	tn := p[1]
	id := p[2]
	table := h.Tables[tn]

	switch r.Method {
	case "GET":
		row := h.DB.QueryRow("select * from `"+tn+"` where "+h.Tables[tn].PrimaryKey.Name+" = ?", id)
		rfmt := table.FormatRows()
		err := row.Scan(rfmt...)

		if err != nil {
			WriteResponse(w, Result{Err: "record not found"}, 404)
			return
		}
		resp := table.ConvertRows(rfmt)

		WriteResponse(w, Result{Resp: map[string]interface{}{"record": resp}}, 200)
		return
	case "POST":
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		params := make(map[string]interface{}, len(table.Fields))
		decoder.Decode(&params)
		err := table.ValidateParams(params)

		if err != nil {
			WriteResponse(w, Result{Err: err.Error()}, 400)
			return
		}

		fields := make([]string, 0)
		rowsToUpdate := make([]interface{}, 0)

		for k, p := range params {
			if k == table.PrimaryKey.Name {
				WriteResponse(w, Result{Err: "field " + k + " have invalid type"}, 400)
				return
			}
			fields = append(fields, fmt.Sprintf("`%s` = ?", k))
			rowsToUpdate = append(rowsToUpdate, p)
		}

		rowsToUpdate = append(rowsToUpdate, id)
		res, err := h.DB.Exec(
			fmt.Sprintf("update `%s` set %s where `%s` = ? ", tn, strings.Join(fields, ", "), table.PrimaryKey.Name),
			rowsToUpdate...,
		)
		if err != nil {
			WriteResponse(w, Result{}, 500)
			return
		}

		rowsAffected, _ := res.RowsAffected()
		WriteResponse(w, Result{Resp: map[string]interface{}{"updated": rowsAffected}}, 200)
		return
	case "DELETE":
		res, err := h.DB.Exec(fmt.Sprintf("delete from `%s` where `%s` = ? ", tn, table.PrimaryKey.Name), id)
		if err != nil {
			WriteResponse(w, Result{}, 500)
			return
		}
		rowsAffected, _ := res.RowsAffected()

		WriteResponse(w, Result{Resp: map[string]interface{}{"deleted": rowsAffected}}, 200)
		return
	default:
		WriteResponse(w, Result{}, 400)
		return
	}
}

func (h *Handler) handleTablesList(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, Result{Resp: map[string]interface{}{"tables": GetTables(h.DB)}}, 200)
}

func GetTables(db *sql.DB) []string {
	var tbls []string
	var t string

	rows, _ := db.Query("SHOW TABLES")
	for rows.Next() {
		rows.Scan(&t)
		tbls = append(tbls, t)
	}
	rows.Close()
	return tbls
}

func (t *Table) FormatRows() []interface{} {
	rows := make([]interface{}, len(t.Fields))
	for i, field := range t.Fields {
		switch field.Type {
		case "varchar", "text":
			rows[i] = new(sql.NullString)
		case "int":
			rows[i] = new(sql.NullInt64)
		}
	}
	return rows
}

func (t *Table) FormatValsToInsert(params map[string]interface{}) []interface{} {
	valsToInsert := make([]interface{}, len(t.Fields))
	for i, field := range t.Fields {
		if t.PrimaryKey.Name == field.Name && t.PrimaryKey.Auto == true {
			continue
		}
		if _, ok := params[field.Name]; !ok {
			if !field.IsRequired {
				valsToInsert[i] = nil
			} else {
				var f interface{}
				switch field.Type {
				case "varchar", "text":
					f = ""
				case "int":
					f = 0
				default:
					f = nil
				}
				valsToInsert[i] = f
			}
		} else {
			valsToInsert[i] = params[field.Name]
		}
	}
	return valsToInsert
}

func (t *Table) ConvertRows(rows []interface{}) map[string]interface{} {
	namedRows := make(map[string]interface{}, len(rows))
	for i, r := range rows {
		switch r.(type) {
		case *sql.NullInt64:
			if v, ok := r.(*sql.NullInt64); ok {
				if v.Valid {
					namedRows[t.Fields[i].Name] = v.Int64
				} else {
					namedRows[t.Fields[i].Name] = nil
				}
			}
		case *sql.NullString:
			if v, ok := r.(*sql.NullString); ok {
				if v.Valid {
					namedRows[t.Fields[i].Name] = v.String
				} else {
					namedRows[t.Fields[i].Name] = nil
				}

			}
		}
	}
	return namedRows
}

func (t *Table) ValidateParams(params map[string]interface{}) error {
	for _, field := range t.Fields {
		if f, ok := params[field.Name]; ok {
			switch f.(type) {
			case float64, int:
				if field.Type != "int" {
					return fmt.Errorf("field %s have invalid type", field.Name)
				}
			case string:
				if field.Type != "varchar" && field.Type != "text" {
					return fmt.Errorf("field %s have invalid type", field.Name)
				}
			case nil:
				if field.IsRequired == true {
					return fmt.Errorf("field %s have invalid type", field.Name)
				}
			default:
				return fmt.Errorf("field %s have invalid type", field.Name)
			}
		}
	}
	return nil
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	tables := GetTables(db)
	handler := Handler{DB: db, Tables: map[string]Table{}, DefaultLimit: "5", DefaultOffset: "0"}

	for _, t := range tables {
		var ti TableInfo
		var fields []Field
		var primaryKey PrimaryKey

		columns, _ := db.Query("SHOW FULL COLUMNS FROM " + t)

		for columns.Next() {
			columns.Scan(&ti.Field, &ti.Type, &ti.Collation, &ti.Null, &ti.Key,
				&ti.Default, &ti.Extra, &ti.Privileges, &ti.Comment)

			if strings.Contains(ti.Type, "(") {
				ti.Type = ti.Type[:strings.Index(ti.Type, "(")]
			}

			if ti.Key == "PRI" {
				primaryKey = PrimaryKey{ti.Field, false}
				if ti.Extra == "auto_increment" {
					primaryKey.Auto = true
				}
			}

			isRequired := true
			if ti.Null == "YES" {
				isRequired = false
			}

			fields = append(fields, Field{ti.Field, ti.Type, isRequired})
		}

		handler.Tables[t] = Table{fields, primaryKey}
		columns.Close()
	}

	return &handler, nil
}
