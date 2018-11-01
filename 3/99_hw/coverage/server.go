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

func (u *XMLUsers) FilterUsers(str string, limit int, offset int) {
	offsetCounter := offset
	newUsers := make([]XMLUser, 0, len(u.Users))

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
	if orderBy != OrderByAsc && orderBy != OrderByDesc && orderBy != OrderByAsIs {
		return fmt.Errorf("ErrorBadOrder")
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
		if orderBy == OrderByAsIs {
			return nil
		}
		By(name).Sort(s.Users)
	case "Id", "id":
		if orderBy == OrderByAsIs {
			return nil
		}
		By(id).Sort(s.Users)
	case "Age", "age":
		if orderBy == OrderByAsIs {
			return nil
		}
		By(age).Sort(s.Users)
	default:
		return fmt.Errorf("ErrorBadOrderField")
	}

	return nil
}

func getData(src string, xmlParsed *XMLUsers, w *http.ResponseWriter) {
	file, _ := ioutil.ReadFile(src)
	xml.Unmarshal(file, &xmlParsed)

	for idx, _ := range xmlParsed.Users {
		xmlParsed.Users[idx].setFullName()
	}
}

type SearchServer struct {
	DataSource string
	AccessToken string
}

type SearchServerInterface interface {
	Search() func(w http.ResponseWriter, r *http.Request)
}

func (srv *SearchServer) Search() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var xmlData XMLUsers

		q := r.URL.Query()
	
		token := r.Header.Get("Accesstoken")
		query := q.Get("query")
		orderBy, _ := strconv.Atoi(q.Get("order_by"))
		offset, _ := strconv.Atoi(q.Get("offset"))
		limit, _ := strconv.Atoi(q.Get("limit"))
		orderField := q.Get("order_field")
	
		if token != srv.AccessToken {
			http.Error(w, http.StatusText(401), 401)
			return
		}
	
		getData(srv.DataSource, &xmlData, &w)
		xmlData.FilterUsers(query, limit, offset)
		err := xmlData.SortUsers(orderField, orderBy)
	
		if err != nil {
			errResp := SearchErrorResponse{err.Error()}
			errRespJson, _ := json.Marshal(errResp)
	
			w.WriteHeader(http.StatusBadRequest)
			w.Write(errRespJson)
			return
		}
		result, _ := json.Marshal(xmlData.Users)
		w.Write(result)
	}
}
