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
	"strconv"
	"regexp"
	"log"
	"os"

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
			log.Printf("Find hash %s\n", hash)
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


func GetDetails(hashes []string) (books []Book) {
	for _, md5 := range hashes {
		apiurl := fmt.Sprintf("http://libgen.io/json.php?md5=%s", md5)
		if r, err := http.Get(apiurl); err == nil {
			defer r.Body.Close()

			if b, err := ioutil.ReadAll(r.Body); err == nil {
				book := ParseResponse(b)
				size, _ := strconv.Atoi(book.Filesize)
				fmt.Printf("\n[%s] %s\n", book.Id, book.Title)
				fmt.Printf("@year [%4s] - ", book.Year)
				fmt.Printf("@author [%10s] ", book.Author)
				fmt.Printf("@size [%8s] ", humanize.Bytes(uint64(size)))
				fmt.Printf("@format [%4s]\n", book.Extension)
				books = append(books, book)
			}
		}
	}

	return
}
