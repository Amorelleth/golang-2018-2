// Code generated. DO NOT EDIT.

package main

import (
	"net/http"
	"fmt"
	"encoding/json"
	"strconv"
	"context"
)

type Result struct {
    Response interface{} `json:"response,omitempty"`
    Err string           `json:"error"`
}

func WriteResponse (w http.ResponseWriter, res Result, status int) {
	resp, _ := json.Marshal(res)
	w.WriteHeader(status)
	w.Write(resp)
}

func (p *ProfileParams) GetParams(r *http.Request) ApiError {
    p.Login = r.FormValue("login")
    return ApiError{Err: nil}
}

func (p *ProfileParams) ValidateParams(w http.ResponseWriter, r *http.Request) ApiError {
    if p.Login == "" {
        return ApiError{400, fmt.Errorf("login must me not empty")}
    }
    return ApiError{200, nil}
}

func (p *CreateParams) GetParams(r *http.Request) ApiError {
    p.Login = r.FormValue("login")
    p.Name = r.FormValue("full_name")
    p.Status = r.FormValue("status")
    name, err := strconv.Atoi(r.FormValue("age"))
    if err != nil {
        return ApiError{400, fmt.Errorf("age must be int")}
    }
    p.Age = name
    return ApiError{Err: nil}
}

func (p *CreateParams) ValidateParams(w http.ResponseWriter, r *http.Request) ApiError {
    if p.Login == "" {
        return ApiError{400, fmt.Errorf("login must me not empty")}
    }
    if len(p.Login) < 10 {
        return ApiError{400, fmt.Errorf("login len must be >= 10")}
    }
    if p.Status == "" {
        p.Status = "user"
    }
    if p.Status != "user" && p.Status != "moderator" && p.Status != "admin" {
        return ApiError{400, fmt.Errorf("status must be one of [user, moderator, admin]")}
    }
    if p.Age < 0 {
        return ApiError{400, fmt.Errorf("age must be >= 0")}
    }
    if p.Age > 128 {
        return ApiError{400, fmt.Errorf("age must be <= 128")}
    }
    return ApiError{200, nil}
}

func (h *MyApi) handleProfile(w http.ResponseWriter, r *http.Request) {
    params := ProfileParams{}
    err := params.GetParams(r)
    if err.Err != nil {
        WriteResponse(w, Result{Err: err.Error()}, err.HTTPStatus)
        return
    }
    err = params.ValidateParams(w, r)
    if err.Err != nil {
        WriteResponse(w, Result{Err: err.Error()}, err.HTTPStatus)
        return
    }
    ctx := context.Background()
    res, e := h.Profile(ctx, params)
    if e != nil {
        errApi, ok := e.(ApiError)
        if ok {
            WriteResponse(w, Result{Err: errApi.Error()}, errApi.HTTPStatus)
            return
        }
        WriteResponse(w, Result{Err: e.Error()}, 500)
        return
    }
    WriteResponse(w, Result{Response: res}, 200)
    return
}

func (h *MyApi) handleCreate(w http.ResponseWriter, r *http.Request) {
    params := CreateParams{}
    err := params.GetParams(r)
    if err.Err != nil {
        WriteResponse(w, Result{Err: err.Error()}, err.HTTPStatus)
        return
    }
    err = params.ValidateParams(w, r)
    if err.Err != nil {
        WriteResponse(w, Result{Err: err.Error()}, err.HTTPStatus)
        return
    }
    ctx := context.Background()
    res, e := h.Create(ctx, params)
    if e != nil {
        errApi, ok := e.(ApiError)
        if ok {
            WriteResponse(w, Result{Err: errApi.Error()}, errApi.HTTPStatus)
            return
        }
        WriteResponse(w, Result{Err: e.Error()}, 500)
        return
    }
    WriteResponse(w, Result{Response: res}, 200)
    return
}

func (p *OtherCreateParams) GetParams(r *http.Request) ApiError {
    p.Username = r.FormValue("username")
    p.Name = r.FormValue("account_name")
    p.Class = r.FormValue("class")
    name, err := strconv.Atoi(r.FormValue("level"))
    if err != nil {
        return ApiError{400, fmt.Errorf("level must be int")}
    }
    p.Level = name
    return ApiError{Err: nil}
}

func (p *OtherCreateParams) ValidateParams(w http.ResponseWriter, r *http.Request) ApiError {
    if p.Username == "" {
        return ApiError{400, fmt.Errorf("username must me not empty")}
    }
    if len(p.Username) < 3 {
        return ApiError{400, fmt.Errorf("username len must be >= 3")}
    }
    if p.Class == "" {
        p.Class = "warrior"
    }
    if p.Class != "warrior" && p.Class != "sorcerer" && p.Class != "rouge" {
        return ApiError{400, fmt.Errorf("class must be one of [warrior, sorcerer, rouge]")}
    }
    if p.Level < 1 {
        return ApiError{400, fmt.Errorf("level must be >= 1")}
    }
    if p.Level > 50 {
        return ApiError{400, fmt.Errorf("level must be <= 50")}
    }
    return ApiError{200, nil}
}

func (h *OtherApi) handleCreate(w http.ResponseWriter, r *http.Request) {
    params := OtherCreateParams{}
    err := params.GetParams(r)
    if err.Err != nil {
        WriteResponse(w, Result{Err: err.Error()}, err.HTTPStatus)
        return
    }
    err = params.ValidateParams(w, r)
    if err.Err != nil {
        WriteResponse(w, Result{Err: err.Error()}, err.HTTPStatus)
        return
    }
    ctx := context.Background()
    res, e := h.Create(ctx, params)
    if e != nil {
        errApi, ok := e.(ApiError)
        if ok {
            WriteResponse(w, Result{Err: errApi.Error()}, errApi.HTTPStatus)
            return
        }
        WriteResponse(w, Result{Err: e.Error()}, 500)
        return
    }
    WriteResponse(w, Result{Response: res}, 200)
    return
}

func (h *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path {
    case "/user/profile": 
        h.handleProfile(w, r)
    case "/user/create": 
        if r.Method != "POST" {
            WriteResponse(w, Result{Err: "bad method"}, 406)
            return
        }
        if r.Header.Get("X-Auth") != "100500" {
            WriteResponse(w, Result{Err: "unauthorized"}, 403)
            return
        }
        h.handleCreate(w, r)
    default:
        WriteResponse(w, Result{Err: "unknown method"}, 404)
    }
}

func (h *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path {
    case "/user/create": 
        if r.Method != "POST" {
            WriteResponse(w, Result{Err: "bad method"}, 406)
            return
        }
        if r.Header.Get("X-Auth") != "100500" {
            WriteResponse(w, Result{Err: "unauthorized"}, 403)
            return
        }
        h.handleCreate(w, r)
    default:
        WriteResponse(w, Result{Err: "unknown method"}, 404)
    }
}

