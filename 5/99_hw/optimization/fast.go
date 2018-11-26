package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

//easyjson:json
type User struct {
	Browsers []string
	Email    string
	Name     string
}

func FastSearch(out io.Writer) {
	var foundUsers string
	var browsers []string
	var user User
	var isAndroid bool
	var isMSIE bool

	fileContents, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	lines := bytes.Split(fileContents, []byte("\n"))
	lenLines := len(lines)
	uniqueBrowsers := make([]string, 0, lenLines)
	users := make([]User, 0, lenLines)

	for i, line := range lines {
		user.UnmarshalJSON(line)

		if err != nil {
			panic(err)
		}
		users = append(users, user)
		isAndroid = false
		isMSIE = false
		android := false
		MSIE := false
		browsers = user.Browsers
		email := user.Email
		name := user.Name

		for _, browser := range browsers {
			android = strings.Contains(browser, "Android")
			MSIE = strings.Contains(browser, "MSIE")

			if android || MSIE {
				if android {
					isAndroid = true
				}
				if MSIE {
					isMSIE = true
				}

				skip := false
				for _, u := range uniqueBrowsers {
					if browser == u {
						skip = true
						break
					}
				}
				if !skip {
					uniqueBrowsers = append(uniqueBrowsers, browser)
				}
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		email = strings.Replace(email, "@", " [at] ", -1)

		foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, name, email)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(uniqueBrowsers))
}
