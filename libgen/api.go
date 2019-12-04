// Copyright © 2019 Antoine Chiny <antoine.chiny@inria.fr>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package libgen

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	json "github.com/json-iterator/go"
)

func Search(pattern string, bookNumber int) ([]string, error) {
	BaseUrl := &url.URL{
		Scheme: "http",
		Host:   "gen.lib.rus.ec",
		Path:   "search.php",
	}

	q := BaseUrl.Query()
	q.Set("req", pattern)
	q.Set("lg_topic", "libgen")
	q.Set("open", "0")
	q.Set("view", "simple")
	q.Set("res", string(bookNumber))
	q.Set("phrase", "1")
	q.Set("column", "def")
	BaseUrl.RawQuery = q.Encode()

	res, err := http.Get(BaseUrl.String())
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			// handle error
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, err
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return parseHashes(string(b)), nil
}

func GetDetails(hashes []string) ([]Book, error) {
	var books []Book
	var formatAuthor string
	var fsize string

	for _, hash := range hashes {
		apiUrl := fmt.Sprintf("http://gen.lib.rus.ec/json.php?ids=%s&fields=id,title,author,"+
			"filesize,extension,md5,year", hash)

		r, err := http.Get(apiUrl)
		if err != nil {
			log.Printf("error reaching API: %v", err)
			return nil, err
		}

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading response from API: %v", err)
			return nil, err
		}

		book := parseResponse(b)
		size, err := strconv.Atoi(book.Filesize)
		if err != nil {
			fsize = "NA"
		} else {
			fsize = humanize.Bytes(uint64(size))
		}

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

		if err := r.Body.Close(); err != nil {
			return nil, err
		}
	}

	return books, nil
}

func parseHashes(response string) []string {
	var hashes []string
	re := regexp.MustCompile(SearchHref)
	matches := re.FindAllString(response, -1)

	for _, m := range matches {
		re := regexp.MustCompile(SearchMD5)
		hash := re.FindString(m)
		if len(hash) == 32 {
			hashes = append(hashes, hash)
		}
	}

	return hashes
}

func parseResponse(data []byte) Book {
	var book Book
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

	return book
}

func pFormat(key string, value string, attr color.Attribute, align string) {
	c := color.New(attr).SprintFunc()
	a := fmt.Sprintf("%%%ss ", align)
	s := fmt.Sprintf("@%s "+a, c(key), value)
	fmt.Printf(a, s)
}

func formatTitle(title string) string {
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

	return strings.Join(cache, " ")
}
