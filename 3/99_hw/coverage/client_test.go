package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var (
	DATA_SOURCE  = "dataset.xml"
	ACCESS_TOKEN = "token"
)

func CheckExpectedError(t *testing.T, err error) {
	if err == nil {
		t.Errorf("Expected an error")
	}
}

func CheckUnexpectedError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Unexpected error")
	}
}

func ServerBrokenErrorJson(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	io.WriteString(w, `"Json: "Bad Json"`)
}

func ServerBrokenResultJson(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `Json: "Bad Json"`)
}

func ServerInternalError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func ServerTimeout(w http.ResponseWriter, r *http.Request) {
	time.Sleep(3 * time.Second)
}

func ServerUnknownError(w http.ResponseWriter, r *http.Request) {}

func TestNegativeOffset(t *testing.T) {
	srv := SearchServer{DATA_SOURCE, ACCESS_TOKEN}
	ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(SearchRequest{Offset: -2})
	
	if err == nil || err.Error() == `offset must be > 0` {

	}
	ts.Close()
}

func TestNegativeLimit(t *testing.T) {
	srv := SearchServer{DATA_SOURCE, ACCESS_TOKEN}
	ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(SearchRequest{Limit:  -5})

	if err == nil || err.Error() != "limit must be > 0" {
		t.Error("error")
	}
	ts.Close()
}

func TestBigLimit(t *testing.T) {
	srv := SearchServer{DATA_SOURCE, ACCESS_TOKEN}
	ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(SearchRequest{Limit: 50})

	CheckUnexpectedError(t, err)
	ts.Close()
}

func TestExceededLimit(t *testing.T) {
	srv := SearchServer{DATA_SOURCE, ACCESS_TOKEN}
	ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(SearchRequest{
		Limit: 2,
		Query: "Nulla cillum",
		OrderField: "id",
		OrderBy: 0,
	})

	CheckUnexpectedError(t, err)
	ts.Close()
}

func TestNextPage(t *testing.T) {
	srv := SearchServer{DATA_SOURCE, ACCESS_TOKEN}
	ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(SearchRequest{Query: "cillum"})

	CheckUnexpectedError(t, err)
	ts.Close()
}

func TestInvalidDataSource(t *testing.T) {
	srv := SearchServer{DATA_SOURCE, ""}
	ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(SearchRequest{})

	CheckExpectedError(t, err)
	ts.Close()
}

func TestInvalidToken(t *testing.T) {
	srv := SearchServer{DATA_SOURCE, ""}
	ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
	srch := SearchClient{
		AccessToken: "some another token",
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(SearchRequest{})

	if err.Error() != "Bad AccessToken" {
		t.Error("TestInvalidToken failed")
	}
	ts.Close()
}

func TestInternalServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(ServerInternalError))
	srch := &SearchClient{
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(SearchRequest{})

	if err.Error() != "SearchServer fatal error" {
		t.Error("TestInternalServerError failed")
	}
	ts.Close()
}

func TestBrokenResultJson(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(ServerBrokenResultJson))
	searchClient := &SearchClient{
		URL: ts.URL,
	}
	_, err := searchClient.FindUsers(SearchRequest{})

	CheckExpectedError(t, err)
}

func TestBrokenErrorJson(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(ServerBrokenErrorJson))
	searchClient := &SearchClient{
		URL: ts.URL,
	}
	_, err := searchClient.FindUsers(SearchRequest{})

	CheckExpectedError(t, err)
}

func TestTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(ServerTimeout))
	searchClient := &SearchClient{
		URL: ts.URL,
	}
	_, err := searchClient.FindUsers(SearchRequest{})

	CheckExpectedError(t, err)
	ts.Close()
}

func TestUnknownError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(ServerUnknownError))
	searchClient := &SearchClient{
		URL: "bad_link",
	}

	_, err := searchClient.FindUsers(SearchRequest{})

	CheckExpectedError(t, err)
	ts.Close()
}

func TestBadRequest(t *testing.T) {
	cases := []SearchRequest{
		SearchRequest{
			OrderField: "gender",
		},
		SearchRequest{
			OrderBy: 6,
		},
	}

	for _, sr := range cases {
		srv := SearchServer{DATA_SOURCE, ACCESS_TOKEN}
		ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
		srch := &SearchClient{
			AccessToken: ACCESS_TOKEN,
			URL:         ts.URL,
		}

		_, err := srch.FindUsers(sr)
		CheckExpectedError(t, err)
		ts.Close()
	}
}

func TestSearchServer(t *testing.T) {
	cases := []SearchRequest{
		SearchRequest{Limit: 2, Query:      "cillum", OrderField: "age", OrderBy:    1},
		SearchRequest{Limit: 2, Query:      "cillum", OrderField: "age", OrderBy:    -1},
		SearchRequest{Limit: 2, Query:      "cillum", OrderField: "age", OrderBy:    0},
		SearchRequest{Limit: 2, Query:      "cillum", OrderField: "name", OrderBy:    1},
		SearchRequest{Limit: 2, Query:      "cillum", OrderField: "name", OrderBy:    -1},
		SearchRequest{Limit: 2, Query:      "cillum", OrderField: "id", OrderBy:    -1},
		SearchRequest{Limit: 2, Query:      "cillum", OrderField: "id", OrderBy:    1},
	}

	srv := SearchServer{DATA_SOURCE, ACCESS_TOKEN}
	ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	for _, item := range cases {
		_, err := srch.FindUsers(item)

		CheckUnexpectedError(t, err)
	}
	ts.Close()
}
