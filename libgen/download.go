// Copyright © 2019 Antoine Chiny <antoine.chiny@inria.fr>
// Copyright © 2019 Ryan Ciehanski <ryan@ciehanski.com>
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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/cheggaaa/pb/v3"
)

// DownloadFile grabs the download DownloadURL for the book requested.
// First, it queries Booksdl.org and then b-ok.cc for valid DownloadURL.
// Then, the download process is initiated with a progress bar displayed to
// the user's CLI.
func DownloadFile(book EReadable, outputPath string) error {
	var filesize int64
	filename := generateDownloadFilename(book)

	// create another filename variable, this time stripping out : and / from the filename
	strippedFilename := strings.Replace(filename, ":", "", -1)

	// check to see if the book is already downloaded
	for _, strFilename := range []string{strippedFilename, filename} {
		if _, err := os.Stat(outputPath + "/" + strFilename); err == nil {
			fmt.Printf("%s already downloaded\n", filename)
			return nil
		}
	}

	fmt.Println("Downloading", strippedFilename)
	req, err := http.NewRequest("GET", book.getDownloadURL(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Accept-Encoding", "*")
	client := http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
	r, err := client.Do(req)
	if err != nil {
		return err
	}

	if r.StatusCode == http.StatusOK {
		filesize = r.ContentLength
		bar := pb.Full.Start64(filesize)

		out, err := makeFile(outputPath, filename)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, bar.NewProxyReader(r.Body))
		if err != nil {
			return err
		}

		bar.Finish()

		if err := out.Close(); err != nil {
			return err
		}
		if err := r.Body.Close(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unable to reach mirror %v: HTTP %v", req.Host, r.StatusCode)
	}

	return nil
}

// DownloadDbdump downloads the selected database dump from
// Library Genesis.
func DownloadDbdump(filename string, outputPath string) error {
	filename = RemoveQuotes(filename)
	mirror := GetWorkingMirror(SearchMirrors)
	client := http.Client{Timeout: HTTPClientTimeout, Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
	r, err := client.Get(fmt.Sprintf("%s/dbdumps/%s", mirror.String(), filename))
	if err != nil {
		return err
	}

	if r.StatusCode == http.StatusOK {
		filesize := r.ContentLength
		bar := pb.Full.Start64(filesize)

		out, err := makeFile(outputPath, filename)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, bar.NewProxyReader(r.Body))
		if err != nil {
			return err
		}

		bar.Finish()

		if err := out.Close(); err != nil {
			return err
		}
		if err := r.Body.Close(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unable to reach mirror: HTTP %v", r.StatusCode)
	}

	return nil
}

// GetDownloadURL picks a random download mirror to download the specified
// resource from.
// This is a hack that I don't like and needs to be revisited.
func GetDownloadURL(book *Book) error {
	chosenMirror := DownloadMirrors[rand.Intn(len(DownloadMirrors))]

	var x int
	tries := 3
	for tries >= x {
		switch chosenMirror.Hostname() {
		case "62.182.86.140":
			if err := getLibraryLolURL(book); err != nil {
				if err := getBooksdlDownloadURL(book); err != nil {
					return err

				}
			}
		case "libgen.rocks":
			if err := getBooksdlDownloadURL(book); err != nil {
				if err = getLibraryLolURL(book); err != nil {
					return err
				}
			}
		}
		if book.DownloadURL != "" {
			break
		}
		// Increment tries
		x++
	}

	if book.DownloadURL == "" {
		return fmt.Errorf("unable to retrieve download link for desired resource")
	}
	return nil
}

func getLibraryLolURL(book *Book) error {
	baseURL := &url.URL{
		Scheme: "http",
		Host:   "library.lol",
		Path:   "main/",
	}
	queryURL := baseURL.String() + book.Md5
	book.PageURL = queryURL

	b, err := getBody(queryURL)
	if err != nil {
		return err
	}

	downloadURL := findMatch(libraryLolReg, b)
	if downloadURL == nil {
		return errors.New("no valid download LibraryLol download URL found")
	}

	book.DownloadURL = string(downloadURL)

	return nil
}

func getBooksdlDownloadURL(book *Book) error {
	baseURL := &url.URL{
		Scheme: "https",
		Host:   "cdn1.booksdl.org",
		Path:   "ads.php",
	}
	q := baseURL.Query()
	q.Set("md5", book.Md5)
	baseURL.RawQuery = q.Encode()
	book.PageURL = baseURL.String()

	b, err := getBody(baseURL.String())
	if err != nil {
		return err
	}

	downloadURL := findMatch(booksdlReg, b)
	if downloadURL == nil {
		return errors.New("no valid download Booksdl download URL found")
	}
	book.DownloadURL = fmt.Sprintf("https://libgen.rocks/%s", string(downloadURL))

	return nil
}

func makeFile(outputPath, filename string) (*os.File, error) {
	var out *os.File
	var mkErr error

	// Handle long titles
	if len(filename) >= 256 {
		filename = filename[:256]
	}

	// if output path was not provided
	if outputPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		if stat, err := os.Stat(fmt.Sprintf("%s/libgen", wd)); err == nil && stat.IsDir() {
			out, mkErr = os.Create(fmt.Sprintf("%s/libgen/%s", wd, filename))
		} else {
			if err := os.Mkdir(fmt.Sprintf("%s/libgen", wd), 0755); err != nil {
				return nil, err
			}
			out, mkErr = os.Create(fmt.Sprintf("%s/libgen/%s", wd, filename))
		}
		if mkErr != nil {
			return nil, mkErr
		}
		// If output path was provided
	} else {
		if stat, err := os.Stat(outputPath); err == nil && stat.IsDir() {
			out, err = os.Create(fmt.Sprintf("%s/%s", outputPath, filename))
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New("invalid output path")
		}
	}

	return out, nil
}

// findMatch is a helper function that searches an []byte
// for a specified regex and returns the matches.
func findMatch(reg string, response []byte) []byte {
	re := regexp.MustCompile(reg)
	match := re.FindString(string(response))

	if match != "" {
		return []byte(match)
	}

	return nil
}

type EReadable interface {
	getAuthor() string
	getExtension() string
	getTitle() string
	getDownloadURL() string
}

func generateDownloadFilename(file EReadable) string {
	var tmp []string
	tmp = append(tmp, file.getTitle())
	tmp = append(tmp, fmt.Sprintf(" by %s", file.getAuthor()))
	tmp = append(tmp, fmt.Sprintf(".%s", file.getExtension()))
	fmt.Println("Extension: ", file.getExtension())
	return strings.Join(tmp, "")
}

type ScienceMagazine struct {
	SourceCode  string `json:"source_code"`
	DOI         string `json:"doi"`
	Author      string `json:"author"`
	Title       string `json:"title"`
	Year        string `json:"year"`
	Volume      string `json:"volume"`
	Issue       string `json:"issue"`
	Publisher   string `json:"publisher"`
	DownloadUrl string `json:"download_url"`
	Extension   string `json:"extension"`
}

func (s *ScienceMagazine) getAuthor() string {
	return s.Author
}

func (s *ScienceMagazine) getTitle() string {
	return s.Title
}

func (s *ScienceMagazine) getExtension() string {
	return s.Extension
}

func (s *ScienceMagazine) getDownloadURL() string {
	return s.DownloadUrl
}

// GetScienceMagazineDownload is a helper function that retrieves the Science
// Magazine download URL for a scientific article from libgen.is.
func GetScienceMagazineDownload(doi string) (ScienceMagazine, error) {
	magUrl := fmt.Sprintf("http://library.lol/scimag/%s", doi)
	code, err := getSourceCode(magUrl)
	if err != nil {

	}
	magazine := getMagazine(code, doi)
	return magazine, nil
}

func getMagazine(code string, doi string) ScienceMagazine {
	downloadURL, err := getMagazineDownloadURL(code)
	if err != nil {

	}

	title, err := getMagazineTitle(code)
	if err != nil {

	}

	author, err := getMagazineAuthor(code)
	if err != nil {

	}

	year, err := getMagazineYear(code)
	if err != nil {

	}

	volume, err := getMagazineVolume(code)
	if err != nil {

	}

	issue, err := getMagazineIssue(code)
	if err != nil {

	}

	publisher, err := getMagazinePublisher(code)
	if err != nil {
	}

	filetype, err := getMagazineExtension(downloadURL)
	if err != nil {

	}

	return ScienceMagazine{
		SourceCode:  code,
		DOI:         doi,
		Author:      author,
		Title:       title,
		Year:        year,
		Volume:      volume,
		Issue:       issue,
		Publisher:   publisher,
		DownloadUrl: downloadURL,
		Extension:   filetype,
	}
}

// getSourceCode is a helper function that retrieves the source code for a
// specified URL.
func getSourceCode(site string) (string, error) {
	// send a GET request to the site
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	// curl -X GET http://localhost:3000/auth/dropbox/callback \
	// -u 6xz5d68qbv6s9p2:bzvaika2p87gar6
	req, err := http.NewRequest("GET", site, nil)
	if err != nil {
		return "", errors.New("unable to create request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error sending request for Source Code: ", err)
		return "", errors.New("unable to send request")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body: ", err)
		return "", errors.New("unable to read response body")
	}
	return string(body), nil
}

// getMagazineDownloadURL is a helper function that retrieves the download URL for a
// scientific article from libgen.is, between the link anchor, "GET".
func getMagazineDownloadURL(sourceCode string) (string, error) {
	// get the URL to the link that says "Download"
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	// curl -X GET http://localhost:3000/auth/dropbox/callback \
	// -u 6xz5d68qbv6s9p2:bzvaika2p87gar6
	re := regexp.MustCompile(`<a href="(.*?)">GET</a>`)
	matches := re.FindStringSubmatch(sourceCode)
	if len(matches) == 0 {
		return "", errors.New("no download link found")
	}
	return matches[1], nil
}

func getMagazineTitle(sourceCode string) (string, error) {
	// get the title of the article
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	// curl -X GET http://localhost:3000/auth/dropbox/callback \
	// -u 6xz5d68qbv6s9p2:bzvaika2p87gar6
	re := regexp.MustCompile(`<h1>(.*?)</h1>`)
	matches := re.FindStringSubmatch(sourceCode)
	if len(matches) == 0 {
		return "", errors.New("no title found")
	}

	// strip : from the title
	title := matches[1]
	title = strings.Replace(title, ":", "", -1)
	return title, nil
}

func getMagazineAuthor(sourceCode string) (string, error) {
	// get the author of the article
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	// curl -X GET http://localhost:3000/auth/dropbox/callback \
	// -u 6xz5d68qbv6s9p2:bzvaika2p87gar6
	re := regexp.MustCompile(`<p>Authors: (.*?)</p>`)
	matches := re.FindStringSubmatch(sourceCode)
	if len(matches) == 0 {
		return "", errors.New("no author found")
	}
	return matches[1], nil
}

func getMagazineYear(sourceCode string) (string, error) {
	// get the year of the article
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	// curl -X GET http://localhost:3000/auth/dropbox/callback \
	// -u 6xz5d68qbv6s9p2:bzvaika2p87gar6
	re := regexp.MustCompile(`<p>Year: (.*?)</p>`)
	matches := re.FindStringSubmatch(sourceCode)
	if len(matches) == 0 {
		return "", errors.New("no year found")
	}
	return matches[1], nil
}

func getMagazineVolume(sourceCode string) (string, error) {
	// get the volume of the article
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	// curl -X GET http://localhost:3000/auth/dropbox/callback \
	// -u 6xz5d68qbv6s9p2:bzvaika2p87gar6
	re := regexp.MustCompile(`<p>Volume: (.*?)</p>`)
	matches := re.FindStringSubmatch(sourceCode)
	if len(matches) == 0 {
		return "", errors.New("no volume found")
	}
	return matches[1], nil
}

func getMagazineIssue(sourceCode string) (string, error) {
	// get the issue of the article
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	// curl -X GET http://localhost:3000/auth/dropbox/callback \
	// -u 6xz5d68qbv6s9p2:bzvaika2p87gar6
	re := regexp.MustCompile(`<p>Issue: (.*?)</p>`)
	matches := re.FindStringSubmatch(sourceCode)
	if len(matches) == 0 {
		return "", errors.New("no issue found")
	}
	return matches[1], nil
}

func getMagazinePages(sourceCode string) (string, error) {
	// get the pages of the article
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	// curl -X GET http://localhost:3000/auth/dropbox/callback \
	// -u 6xz5d68qbv6s9p2:bzvaika2p87gar6
	re := regexp.MustCompile(`<p>Pages: (.*?)</p>`)
	matches := re.FindStringSubmatch(sourceCode)
	if len(matches) == 0 {
		return "", errors.New("no pages found")
	}
	return matches[1], nil
}

func getMagazinePublisher(sourceCode string) (string, error) {
	// get the publisher of the article
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	// curl -X GET http://localhost:3000/auth/dropbox/callback \
	// -u 6xz5d68qbv6s9p2:bzvaika2p87gar6
	re := regexp.MustCompile(`<p>Publisher: (.*?)</p>`)
	matches := re.FindStringSubmatch(sourceCode)
	if len(matches) == 0 {
		return "", errors.New("no publisher found")
	}
	return matches[1], nil
}

func getMagazineExtension(downloadUrl string) (string, error) {
	// get the filetype of the article
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	// curl -X GET http://localhost:3000/auth/dropbox/callback \
	// -u 6xz5d68qbv6s9p2:bzvaika2p87gar6
	re := regexp.MustCompile(`\.([^.]*?)$`)
	matches := re.FindStringSubmatch(downloadUrl)
	if len(matches) == 0 {
		return "", errors.New("no filetype found")
	}
	return matches[1], nil
}
