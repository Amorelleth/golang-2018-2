package main

import (
	// "fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type TestCase struct {
	Expected SearchResponse
	SearchClient
	SearchRequest
}

func TestSearchServer(t *testing.T) {
	cases := []TestCase{
		TestCase{
			SearchClient: SearchClient{
				AccessToken: "kek-token",
			},
			SearchRequest: SearchRequest{
				Limit:      3,
				Offset:     5,
				Query:      "Nulla cillum",
				OrderField: "name",
				OrderBy:    1,
			},
			Expected: SearchResponse{
				Users:    []User{},
				NextPage: false,
			},
		},

		TestCase{
			SearchClient: SearchClient{
				AccessToken: "kek-token",
			},
			SearchRequest: SearchRequest{
				Limit: 5,
				// Offset: 1,
				Query: "Nulla cillum",
				// OrderField: "name",
				// OrderBy:    1,
			},
			Expected: SearchResponse{
				Users:    []User{},
				NextPage: false,
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range cases {
		c := &SearchClient{
			AccessToken: item.AccessToken,
			URL:         ts.URL,
		}
		result, err := c.FindUsers(SearchRequest{
			Limit:      item.Limit,
			Offset:     item.Offset,
			Query:      item.Query,
			OrderField: item.OrderField,
			OrderBy:    item.OrderBy,
		})

		if err != nil {
			t.Errorf("[%d] error: %#v", caseNum, err)
		}

		if !reflect.DeepEqual(*result, item.Expected) {
			t.Errorf("results not match\nGot: %v\nExpected: %v", *result, item.Expected)
		}

		// fmt.Println(string(result.Users))
	}
	ts.Close()
}
