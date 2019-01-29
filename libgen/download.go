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
	"io"
	"log"
	"fmt"
	"regexp"
	"path/filepath"

	"gopkg.in/cheggaaa/pb.v1"
)


const (
	SearchHref = "<a href='book/index.php.+</a>"
	SearchId = "book/index\\.php\\?md5=\\w{32}"
	SearchMD5 = "[A-Z0-9]{32}"
	SearchTitle = ">[^<]+"
	SearchUrl = "http://booksdl.org/get\\.php\\?md5=\\w{32}\\&key=\\w{16}"
	NumberOfBooks = "10"
)


type WriteCounter struct {
	Total uint64
	Pb *pb.ProgressBar
}


type BookFile struct {
	size int64
	name string
	path string
	data []byte
}


func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.Pb.Add64(int64(n))
	return n, nil
}


func GetHref(HttpResponse string) (href string) {
	re := regexp.MustCompile(SearchUrl)
	matchs := re.FindAllString(HttpResponse, -1)

	if len(matchs) > 0 {
		href = matchs[0]
	}

	return
}


func GetDownloadUrl(book *Book) error {
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
		return err
	}
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
			return err
		}
		book.Url = GetHref(string(b))
	}
	return nil
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


func GetBookFilename(book Book) (filename string) {
	var tmp []string

	tmp = append(tmp, book.Title)
	tmp = append(tmp, fmt.Sprintf(" (%s - %s)", book.Year, book.Author))
	tmp = append(tmp, fmt.Sprintf(".%s", book.Extension))
	filename = strings.Join(tmp, "")
	return
}


func DownloadBook(book Book) error {
	var (
		filename string
		filesize int64
		counter *WriteCounter
	)

	filename = GetBookFilename(book)
	counter = &WriteCounter{}

	log.Println("Download Started")
	if res, err := http.Get(book.Url); err == nil {
		if res.StatusCode == http.StatusOK {
			defer res.Body.Close()

			filesize = res.ContentLength
			counter.Pb = pb.StartNew(int(filesize))
			out, err := os.Create(filename + ".tmp")

			if err != nil {
				return err
			}
			defer out.Close()
			_, err = io.Copy(out, io.TeeReader(res.Body, counter))
			if err != nil {
				return err
			}
			err = os.Rename(filename + ".tmp", filename)
			if err != nil {
				return err
			}

			log.Printf("[OK] %s\n", filename)
		}
	}
	return nil
}
