// Copyright 2012 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libgen


import (
	"net/http"
	"net/url"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"strings"
	"strconv"
	"regexp"
	"log"
	"os"

	"github.com/fatih/color"
	"github.com/dustin/go-humanize"
)


type Book struct {
	Title string
	Id string
	Author string
	Filesize string
	Extension string
	Md5 string
	Year string
	Url string
}


func ParseHashes(response string) (hashes []string) {
	re := regexp.MustCompile(SearchHref)
	matchs := re.FindAllString(response, -1)

	for _, m := range matchs {
		re := regexp.MustCompile(SearchMD5)
		hash := re.FindString(m)
		if len(hash) == 32 {
			log.Printf("New hash found %s\n", hash)
			hashes = append(hashes, hash)
		}
	}

	return
}

func Search(pattern string, bookNumber int) (hashes []string) {
	BaseUrl := &url.URL{
		Scheme: "http",
		Host: "libgen.io",
		Path: "search.php",
	}

	q := BaseUrl.Query()
	q.Set("req", pattern)
	q.Set("lg_topic", "libgen")
	q.Set("open", "0")
	q.Set("res", string(bookNumber))
	q.Set("phrase", "1")
	q.Set("column", "def")
	BaseUrl.RawQuery = q.Encode()

	res, err := http.Get(BaseUrl.String())
	if err != nil {
		log.Printf("http.Get(%q) error: %v", BaseUrl, err)
		os.Exit(-1)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("res.StatusCode = %d; want %d",
			res.StatusCode,
			http.StatusOK,
		)
		os.Exit(-1)
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("error reading resp body: %v", err)
		os.Exit(-1)
	}
	hashes = ParseHashes(string(b))

	return
}


func ParseResponse(data []byte) (book Book) {
	var cache []map[string]string

	if err := json.Unmarshal(data, &cache); err == nil {
		for _, item := range cache {
			for k, v := range item {
				switch k {
				case "id":
					book.Id = v
				case "title":
					book.Title = v
				case "author":
					book.Author = v
				case "filesize":
					book.Filesize = v
				case "extension":
					book.Extension = v
				case "md5":
					book.Md5 = v
				case "year":
					book.Year = v
				}
			}
		}
	}

	return
}

func formatTitle(title string) (formatTitle string) {
	var cache []string
	var counter int

	if len(title) < 60 {
		return title
	}

	title = strings.TrimSpace(title)
	for _, t := range strings.Split(title, " ") {
		counter += len(t)

		if counter > 60 {
			counter = 0
			t = t + "\n"
		}
		cache = append(cache, t)
	}
	formatTitle = strings.Join(cache, " ")

	return
}

func pFormat(key string, value string, attr color.Attribute, align string) {
	c := color.New(attr).SprintFunc()
	a := fmt.Sprintf("%%%ss ", align)
	s := fmt.Sprintf("@%s "+a, c(key), value)
	fmt.Printf(a, s)
}

func GetDetails(hashes []string) (books []Book) {
	var formatAuthor string

	for _, md5 := range hashes {
		apiurl := fmt.Sprintf("http://libgen.io/json.php?md5=%s", md5)
		if r, err := http.Get(apiurl); err == nil {
			defer r.Body.Close()

			if b, err := ioutil.ReadAll(r.Body); err == nil {
				book := ParseResponse(b)
				size, _ := strconv.Atoi(book.Filesize)
				fsize := humanize.Bytes(uint64(size))

				fmt.Println(strings.Repeat("-", 80))
				fTitle := fmt.Sprintf("%5s %s", book.Id, book.Title)
				fTitle = formatTitle(fTitle)
				fmt.Printf("%s\n    ++ ", fTitle)
				if len(book.Author) > 25 {
					formatAuthor = book.Author[:25]
				} else {
					formatAuthor = book.Author
				}
				pFormat("author", formatAuthor, color.FgYellow, "-25")
				pFormat("year", book.Year, color.FgCyan, "4")
				pFormat("size", fsize, color.FgGreen, "6")
				pFormat("type", book.Extension, color.FgRed, "4")
				fmt.Println()
				books = append(books, book)
			}
		}
	}

	return
}
