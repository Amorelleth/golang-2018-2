package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"

	"sort"
	"strconv"
	"strings"
)

const (
	DATA_SOURCE         = "dataset.xml"
	DEFAULT_ORDER_FIELD = "name"
)

type XMLUser struct {
	Id        int    `xml:"id" json:"Id"`
	Age       int    `xml:"age" json:"Age"`
	About     string `xml:"about" json:"About"`
	Gender    string `xml:"gender" json:"Gender"`
	LastName  string `xml:"last_name" json:"-"`
	FirstName string `xml:"first_name" json:"-"`
	Name      string `xml:"-" json:"Name"`
}

type XMLUsers struct {
	Users []XMLUser `xml:"row" json"-"`
}

func (u *XMLUser) setFullName() {
	u.Name = u.FirstName + " " + u.LastName
}

type By func(u1, u2 *XMLUser) bool

func (by By) Sort(XMLUsers []XMLUser) {
	ps := &XMLUserSorter{
		XMLUsers: XMLUsers,
		by:       by,
	}
	sort.Sort(ps)
}

type XMLUserSorter struct {
	XMLUsers []XMLUser
	by       func(u1, u2 *XMLUser) bool
}

func (s *XMLUserSorter) Len() int {
	return len(s.XMLUsers)
}

func (s *XMLUserSorter) Swap(i, j int) {
	s.XMLUsers[i], s.XMLUsers[j] = s.XMLUsers[j], s.XMLUsers[i]
}

func (s *XMLUserSorter) Less(i, j int) bool {
	return s.by(&s.XMLUsers[i], &s.XMLUsers[j])
}

// const (
// 	OrderByAsc  = -1
// 	OrderByAsIs = 0
// 	OrderByDesc = 1

// 	ErrorBadOrderField = `OrderField invalid`
// )

func (u *XMLUsers) FilterUsers(str string, limit int, offset int) {
	offsetCounter := offset

	newUsers := make([]XMLUser, 0, len(u.Users))

	// fmt.Println(len(u.Users))

	for _, user := range u.Users {
		if strings.Contains(user.Name, str) || strings.Contains(user.About, str) {
			if len(newUsers) < limit && offsetCounter <= 0 {
				newUsers = append(newUsers, user)
			} else {
				offsetCounter--
			}
		}
	}
	u.Users = newUsers
}

func (s *XMLUsers) SortUsers(orderField string, orderBy int) error {
	if orderBy == OrderByAsIs {
		return nil
	}

	name := func(u1, u2 *XMLUser) bool {
		if orderBy == OrderByDesc {
			return u1.Name < u2.Name
		}
		return u1.Name > u2.Name
	}

	id := func(u1, u2 *XMLUser) bool {
		if orderBy == OrderByDesc {
			return u1.Id < u2.Id
		}
		return u1.Id > u2.Id
	}

	age := func(u1, u2 *XMLUser) bool {
		if orderBy == OrderByDesc {
			return u1.Age < u2.Age
		}
		return u1.Age > u2.Age
	}

	switch orderField {
	case "Name", "name", "":
		By(name).Sort(s.Users)
	case "Id", "id":
		By(id).Sort(s.Users)
	case "Age", "age":
		By(age).Sort(s.Users)
	default:
		fmt.Errorf("some error")
	}

	return nil
}

func getData(src string, xmlParsed *XMLUsers) {
	file, _ := ioutil.ReadFile(src)
	xml.Unmarshal(file, &xmlParsed)

	for idx, _ := range xmlParsed.Users {
		xmlParsed.Users[idx].setFullName()
	}
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	var xmlData XMLUsers

	q := r.URL.Query()

	token := r.Header["Accesstoken"][0]

	query := q["query"][0]                       //
	orderBy, _ := strconv.Atoi(q["order_by"][0]) //
	offset, _ := strconv.Atoi(q["offset"][0])
	limit, _ := strconv.Atoi(q["limit"][0])
	orderField := q["order_field"][0] //

	if token != "kek-token" {
		w.WriteHeader(http.StatusUnauthorized)
	}

	getData(DATA_SOURCE, &xmlData)

	xmlData.FilterUsers(query, limit, offset)
	xmlData.SortUsers(orderField, orderBy)

	// fmt.Println(xmlData)

	result, _ := json.Marshal(xmlData.Users)
	fmt.Println(string(result))

	w.Write(result)
	// w.WriteHeader(http.StatusInternalServerError)
}

// func main() {
// 	mux := http.NewServeMux()
// 	// mux.HandleFunc("/test", SearchServer)
// 	mux.HandleFunc("/", SearchServer)

// 	server := http.Server{
// 		Addr:         "localhost:8080",
// 		Handler:      mux,
// 		ReadTimeout:  10 * time.Second,
// 		WriteTimeout: 10 * time.Second,
// 	}

// 	fmt.Println("starting server at :8080")
// 	server.ListenAndServe()
// }
