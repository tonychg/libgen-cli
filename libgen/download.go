// Copyright 2012 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libgen


import (
	"net/http"
	"net/url"
	"io/ioutil"
	"strings"
	"os"
	"log"
	"regexp"
	"path/filepath"
)


const (
	SearchHref = "<a href='book/index.php.+</a>"
	SearchId = "book/index\\.php\\?md5=\\w{32}"
	SearchMD5 = "[A-Z0-9]{32}"
	SearchTitle = ">[^<]+"
	SearchUrl = "http://booksdl.org/get\\.php\\?md5=\\w{32}\\&key=\\w{16}"
	NumberOfBooks = "10"
)


type BookFile struct {
	size int64
	name string
	path string
	data []byte
}


func GetHref(HttpResponse string) (href string) {
	re := regexp.MustCompile(SearchUrl)
	matchs := re.FindAllString(HttpResponse, -1)

	if len(matchs) > 0 {
		href = matchs[0]
	}

	return
}


func GetDownloadUrl(book Book) (downloadUrl string) {
	BaseUrl := &url.URL{
		Scheme: "http",
		Host: "booksdescr.org",
		Path: "ads.php",
	}

	q := BaseUrl.Query()
	q.Set("md5", book.Md5)
	BaseUrl.RawQuery = q.Encode()

	res, err := http.Get(BaseUrl.String())
	if err != nil {
		log.Printf("http.Get(%q) error: %v", BaseUrl, err)
	} else {
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			log.Printf("res.StatusCode = %d; want %d",
				res.StatusCode,
				http.StatusOK,
			)
		} else {
			b, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Printf("error reading resp body: %v", err)
			} else {
				downloadUrl = GetHref(string(b))
			}
		}
	}
	return
}


func WriteFile(book BookFile, dest string) (bytes int) {
	bytes = 0

	log.Println("Start writing")
	p := filepath.Join(dest, book.name)
	if f, err := os.Create(p); err == nil {
		defer f.Close()
		if bytes, err = f.Write(book.data); err == nil {
			log.Printf("%d written bytes\n", bytes)
			f.Sync()
			return
		}
	}

	log.Println("Fail to write new file\n")
	return
}


func DownloadBook(url string) (bytes int ) {
	var bookfile BookFile

	log.Printf("Downloading %s\n", url)
	if res, err := http.Get(url); err == nil {
		if res.StatusCode == http.StatusOK {
			defer res.Body.Close()

			re := regexp.MustCompile("\".+\"")

			filename := res.Header.Get("Content-Disposition")
			filename = re.FindString(filename)

			bookfile.size = res.ContentLength
			bookfile.name = strings.Replace(filename, "\"", "", -1)

			log.Printf("[OK] %s\n", bookfile.name)

			if b, err := ioutil.ReadAll(res.Body); err == nil {
				bookfile.data = b
				bytes = WriteFile(bookfile, "./")
			}
		}
	}
	return
}
