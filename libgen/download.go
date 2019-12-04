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
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/cheggaaa/pb/v3"
)

func DownloadBook(book Book) error {
	var filesize int64
	filename := getBookFilename(book)

	log.Printf("Download started for %s\n", book.Title)

	err := getDownloadUrl(&book)
	if err != nil {
		return err
	}
	if book.Url == "" {
		return fmt.Errorf("unable to retrieve download link for book")
	}

	res, err := http.Get(book.Url)
	if err != nil {
		return err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			// handle error
		}
	}()

	if res.StatusCode == http.StatusOK {
		filesize = res.ContentLength
		bar := pb.Full.Start64(filesize)

		out, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer func() {
			if err := out.Close(); err != nil {
				// handle err
			}
		}()

		_, err = io.Copy(out, bar.NewProxyReader(res.Body))
		if err != nil {
			return err
		}

		bar.Finish()

		log.Printf("[OK] %s\n", filename)
	} else {
		return fmt.Errorf("unable to reach mirror: %v", res.StatusCode)
	}

	return nil
}

func getDownloadUrl(book *Book) error {
	var err error
	BaseUrl := &url.URL{
		Scheme: "http",
		Host:   "libgen.lc",
		Path:   "ads.php",
	}

	q := BaseUrl.Query()
	q.Set("md5", book.Md5)
	BaseUrl.RawQuery = q.Encode()

	res, err := http.Get(BaseUrl.String())
	if err != nil {
		log.Printf("http.Get(%q) error: %v", BaseUrl, err)
		return err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			// handle error
		}
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to connect to mirror: %v", res.StatusCode)
	} else {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		book.Url = getHref(string(b))
	}

	return nil
}

func getHref(HttpResponse string) string {
	re := regexp.MustCompile(searchUrl)
	matches := re.FindAllString(HttpResponse, -1)

	if len(matches) > 0 {
		return matches[0]
	} else {
		return ""
	}
}

func getBookFilename(book Book) string {
	var tmp []string
	tmp = append(tmp, book.Title)
	tmp = append(tmp, fmt.Sprintf(" (%s - %s)", book.Year, book.Author))
	tmp = append(tmp, fmt.Sprintf(".%s", book.Extension))
	return strings.Join(tmp, "")
}
