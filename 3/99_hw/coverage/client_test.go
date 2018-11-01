package main

import (
	// "fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

type TestCase struct {
	SearchRequest
	Expected SearchResponse
}

func TestNegativeParams(t *testing.T) {
	testCase := TestCase{
		SearchRequest: SearchRequest{
			Limit:  5,
			Offset: -2,
		},
		Expected: SearchResponse{
			Users:    []User{},
			NextPage: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(testCase.SearchRequest)

	if err == nil {
		t.Errorf("Expected an error")
	}

	ts.Close()
}

func TestNegativeLimit(t *testing.T) {
	testCase := TestCase{
		SearchRequest: SearchRequest{
			Limit:  -5,
			Offset: 2,
		},
		Expected: SearchResponse{
			Users:    []User{},
			NextPage: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(testCase.SearchRequest)

	if err == nil {
		t.Errorf("Expected an error")
	}

	ts.Close()
}

func TestExceededLimit(t *testing.T) {
	testCase := TestCase{
		SearchRequest: SearchRequest{
			Query: "Nulla cillum",
			Limit: 50,
		},
		Expected: SearchResponse{
			Users: []User{
				User{
					0, "Boyd Wolf", 22, "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n", "male",
				},
			},
			NextPage: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	result, err := srch.FindUsers(testCase.SearchRequest)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(*result, testCase.Expected) {
		t.Errorf("Got %v\nExpected %v\n", *result, testCase.Expected)
	}

	ts.Close()
}

func TestNextPage(t *testing.T) {
	testCase := TestCase{
		SearchRequest: SearchRequest{
			Query: "Nulla cillum",
		},
		Expected: SearchResponse{
			Users:    []User{},
			NextPage: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	result, err := srch.FindUsers(testCase.SearchRequest)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(*result, testCase.Expected) {
		t.Errorf("Got %v\nExpected %v\n", *result, testCase.Expected)
	}

	ts.Close()
}

func TestWrong(t *testing.T) {
	defer func() {
		DATA_SOURCE = "dataset.xml"
	}()

	DATA_SOURCE = "dataset_.xml"
	testCase := TestCase{
		SearchRequest: SearchRequest{
			Query: "Nulla cillum",
		},
		Expected: SearchResponse{
			Users:    []User{},
			NextPage: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	srch := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(testCase.SearchRequest)

	if err == nil {
		t.Errorf("Expected an error")
	}

	ts.Close()
}

func TestInvalidToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	srch := &SearchClient{
		AccessToken: "some_another_token",
		URL:         ts.URL,
	}

	_, err := srch.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Expected an error")
	}

	ts.Close()
}

func TestBadRequest(t *testing.T) {
	cases := []TestCase{
		TestCase{
			SearchRequest: SearchRequest{
				OrderField: "gender",
			},
			Expected: SearchResponse{},
		},
		TestCase{
			SearchRequest: SearchRequest{
				OrderBy: 6,
			},
			Expected: SearchResponse{},
		},
	}

	for caseNum, testCase := range cases {
		ts := httptest.NewServer(http.HandlerFunc(SearchServer))
		srch := &SearchClient{
			AccessToken: ACCESS_TOKEN,
			URL:         ts.URL,
		}

		_, err := srch.FindUsers(testCase.SearchRequest)

		if err == nil {
			t.Errorf("Case [%d]. Expected an error %v", caseNum, err)
		}

		ts.Close()
	}
}

func TestSearchServer(t *testing.T) {
	cases := []TestCase{
		TestCase{
			SearchRequest: SearchRequest{
				// Limit: 5,
				Offset:     2,
				Query:      "Nulla cillum",
				OrderField: "name",
				OrderBy:    1,
			},
			Expected: SearchResponse{
				Users:    []User{},
				NextPage: false,
			},
		},
		// TestCase{
		// 	SearchClient: SearchClient{
		// 		AccessToken: ACCESS_TOKEN,
		// 	},
		// 	SearchRequest: SearchRequest{
		// 		Limit: 3,
		// 		OrderField: "id",
		// 		OrderBy: -1,
		// 	},
		// 	Expected: SearchResponse{
		// 		Users: []User{},
		// 		NextPage: false,
		// 	},
		// },
		// TestCase{
		// 	SearchClient: SearchClient{
		// 		AccessToken: ACCESS_TOKEN,
		// 	},
		// 	SearchRequest: SearchRequest{
		// 		Limit: 3,
		// 		OrderField: "age",
		// 		OrderBy: 0,
		// 	},
		// 	Expected: SearchResponse{
		// 		Users: []User{},
		// 		NextPage: false,
		// 	},
		// },
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range cases {
		srch := &SearchClient{
			AccessToken: ACCESS_TOKEN,
			URL:         ts.URL,
		}
		result, err := srch.FindUsers(item.SearchRequest)

		if err != nil {
			t.Errorf("[%d] error: %#v", caseNum, err)
		}
		if !reflect.DeepEqual(*result, item.Expected) {
			t.Errorf("Got %v\nExpected %v\n", *result, item.Expected)
		}
		// fmt.Println(result)
	}
	ts.Close()
}

func TestBrokenJson(t *testing.T) {
	cases := []TestCase{
		TestCase{
			SearchRequest: SearchRequest{
				OrderField: "__broken_result_json",
			},
		},
		TestCase{
			SearchRequest: SearchRequest{
				OrderField: "__broken_error_json",
			},
		},
		TestCase{
			SearchRequest: SearchRequest{
				OrderField: "__timeout",
			},
		},
	}

	FakeSearchServer := func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("order_field")

		switch key {
		case "__broken_result_json":
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"Id": "0", "FirstName: "Boyd", "SecondName": "Wolf", "Age": "22", "About": "Nulla cillum", "Gender": "male"}`)
		case "__broken_error_json":
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"Error": ,}`)
		}
	}

	ts := httptest.NewServer(http.HandlerFunc(FakeSearchServer))

	for caseNum, item := range cases {
		c := &SearchClient{
			AccessToken: ACCESS_TOKEN,
			URL:         ts.URL,
		}

		_, err := c.FindUsers(item.SearchRequest)

		if err == nil {
			t.Errorf("Case [%d]: expected an error", caseNum)
		}
	}
	ts.Close()
}

func TestTimeout(t *testing.T) {
	testCase := TestCase{
		SearchRequest: SearchRequest{
			OrderField: "__timeout",
		},
	}

	FakeSearchServer := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
	}

	ts := httptest.NewServer(http.HandlerFunc(FakeSearchServer))

	c := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := c.FindUsers(testCase.SearchRequest)

	if err == nil {
		t.Errorf("Expected an error")
	}

	ts.Close()
}

func TestError(t *testing.T) {
	testCase := TestCase{
		SearchRequest: SearchRequest{
			OrderField: "__timeout",
		},
	}

	FakeSearchServer := func(w http.ResponseWriter, r *http.Request) {

	}

	ts := httptest.NewServer(http.HandlerFunc(FakeSearchServer))

	c := &SearchClient{
		AccessToken: ACCESS_TOKEN,
		URL:         ts.URL,
	}

	_, err := c.FindUsers(testCase.SearchRequest)

	if err == nil {
		t.Errorf("Expected an error")
	}

	ts.Close()
}
