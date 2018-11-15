package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	// "regexp"
	"strings"
	// "log"
)

// BenchmarkSolution-8 500 2782432 ns/op 559910 B/op 10422 allocs/op

// const filePath string = "./data/users.txt"

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}


	lines := strings.Split(string(fileContents), "\n")
	lenLines := len(lines)

	seenBrowsers := make([]string, 0, lenLines) 
	// uniqueBrowsers := 0
	foundUsers := ""


	users := make([]map[string]interface{}, 0, lenLines)
	for _, line := range lines {
		user := make(map[string]interface{})
		// fmt.Printf("%v %v\n", err, line)
		err := json.Unmarshal([]byte(line), &user)
		if err != nil {
			panic(err)
		}
		users = append(users, user)
	}


	for i, user := range users {

		isAndroid := false
		isMSIE := false

		browsers, ok := user["browsers"].([]interface{})
		_, ok2 := user["browsers"].(map[int]string)
fmt.Println(ok2)

		if !ok {
			// log.Println("cant cast browsers")
			continue
		}

		for _, browserRaw := range browsers {
			browser, ok := browserRaw.(string)
			if !ok {
				// log.Println("cant cast browser to string")
				continue
			}
			if strings.Contains(browser, "Android") {
				isAndroid = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers = append(seenBrowsers, browser)
					// uniqueBrowsers++
				}
			}
			if strings.Contains(browser, "MSIE") {
				isMSIE = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers = append(seenBrowsers, browser)
					// uniqueBrowsers++
				}
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}
		email := strings.Replace(user["email"].(string), "@", " [at] ", -1)

		foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user["name"], email)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}


func main() {
	FastSearch(ioutil.Discard)
}
