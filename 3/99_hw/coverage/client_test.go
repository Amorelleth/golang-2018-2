package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type TestCase struct {
	SearchRequest
	Expected SearchResponse
}

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

func ServerBadJson(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `"Json": "Bad Json"`)
}

func ServerTimeout(w http.ResponseWriter, r *http.Request) {
	time.Sleep(3 * time.Second)
}

func ServerUnknownError(w http.ResponseWriter, r *http.Request) {}

func TestNegativeParams(t *testing.T) {
	testCase := TestCase{
		SearchRequest: SearchRequest{
			Limit:  5,
			Offset: -2,
		},
	}

	srv := SearchServer{"dataset.xml"}
	ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(testCase.SearchRequest)

	CheckExpectedError(t, err)
	ts.Close()
}

func TestNegativeLimit(t *testing.T) {
	testCase := TestCase{
		SearchRequest: SearchRequest{
			Limit:  -5,
			Offset: 2,
		},
	}

	srv := SearchServer{"dataset.xml"}
	ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(testCase.SearchRequest)

	CheckExpectedError(t, err)
	ts.Close()
}

func TestExceededLimit(t *testing.T) {
	testCase := TestCase{
		SearchRequest: SearchRequest{
			Query: "Nulla cillum",
			Limit: 50,
		},
	}

	srv := SearchServer{"dataset.xml"}
	ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(testCase.SearchRequest)

	CheckUnexpectedError(t, err)
	ts.Close()
}

func TestNextPage(t *testing.T) {
	testCase := TestCase{
		SearchRequest: SearchRequest{
			Query: "Nulla cillum",
		},
	}

	srv := SearchServer{"dataset.xml"}
	ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(testCase.SearchRequest)

	CheckUnexpectedError(t, err)
	ts.Close()
}

func TestWrong(t *testing.T) {
	srv := SearchServer{"some_another_data.xml"}
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
	srv := SearchServer{"dataset.xml"}
	ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
	srch := &SearchClient{
		AccessToken: "some another token",
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(SearchRequest{})

	CheckExpectedError(t, err)
	ts.Close()
}

func TestBadRequest(t *testing.T) {
	cases := []TestCase{
		TestCase{
			SearchRequest: SearchRequest{
				OrderField: "gender",
			},
		},
		TestCase{
			SearchRequest: SearchRequest{
				OrderBy: 6,
			},
		},
	}

	for caseNum, testCase := range cases {
		srv := SearchServer{"dataset.xml"}
		ts := httptest.NewServer(http.HandlerFunc(srv.Search()))
		srch := &SearchClient{
			AccessToken: ACCESS_TOKEN,
			URL:         ts.URL,
		}

		_, err := srch.FindUsers(testCase.SearchRequest)

		CheckExpectedError(t, err)
		ts.Close()
	}
}

// func TestSearchServer(t *testing.T) {
// 	cases := []TestCase{
// 		TestCase{
// 			SearchRequest: SearchRequest{
// 				// Limit: 5,
// 				Offset:     2,
// 				Query:      "Nulla cillum",
// 				OrderField: "name",
// 				OrderBy:    1,
// 			},
// 			Expected: SearchResponse{
// 				Users:    []User{},
// 				NextPage: false,
// 			},
// 		},
// 		// TestCase{
// 		// 	SearchClient: SearchClient{
// 		// 		AccessToken: ACCESS_TOKEN,
// 		// 	},
// 		// 	SearchRequest: SearchRequest{
// 		// 		Limit: 3,
// 		// 		OrderField: "id",
// 		// 		OrderBy: -1,
// 		// 	},
// 		// 	Expected: SearchResponse{
// 		// 		Users: []User{},
// 		// 		NextPage: false,
// 		// 	},
// 		// },
// 		// TestCase{
// 		// 	SearchClient: SearchClient{
// 		// 		AccessToken: ACCESS_TOKEN,
// 		// 	},
// 		// 	SearchRequest: SearchRequest{
// 		// 		Limit: 3,
// 		// 		OrderField: "age",
// 		// 		OrderBy: 0,
// 		// 	},
// 		// 	Expected: SearchResponse{
// 		// 		Users: []User{},
// 		// 		NextPage: false,
// 		// 	},
// 		// },
// 	}

// 	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

// for caseNum, item := range cases {
// 	srch := &SearchClient{
// 		AccessToken: ACCESS_TOKEN,
// 		URL:         ts.URL,
// 	}
// 	result, err := srch.FindUsers(item.SearchRequest)

// 	if err != nil {
// 		t.Errorf("[%d] error: %#v", caseNum, err)
// 	}
// 	if !reflect.DeepEqual(*result, item.Expected) {
// 		t.Errorf("Got %v\nExpected %v\n", *result, item.Expected)
// 	}
// fmt.Println(result)
// 	}
// 	ts.Close()
// }

func TestBrokenJson(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(ServerBadJson))
	searchClient := &SearchClient{
		URL: ts.URL,
	}
	_, err := searchClient.FindUsers(SearchRequest{})

	if err.Error() != `cant unpack result json: invalid character ':' after top-level value` {
		t.Error("Bad json test :(")
	}
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

	if err == nil {
		t.Error("TestUnknownError :(")
	}

	ts.Close()
}
