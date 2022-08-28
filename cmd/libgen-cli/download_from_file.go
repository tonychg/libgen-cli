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

package libgen_cli

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ciehanski/libgen-cli/libgen"
)

// readInMd5s reads in a file of MD5s and returns an array of MD5s
//  for libgen books
func readInMd5s(fileName string) ([]string, error) {
	fileBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// remove brackets, quotes, and commas from the string file
	fileString := strings.ToLower(string(fileBytes))
	fileString = strings.Replace(fileString, "[", "", -1)
	fileString = strings.Replace(fileString, "]", "", -1)
	fileString = strings.Replace(fileString, "\"", "", -1)
	fileString = strings.Replace(fileString, ",", "", -1)
	fileString = strings.Replace(fileString, " ", "", -1)

	// split the string into an array of md5s
	md5Data := strings.Split(fileString, "\n")

	re := regexp.MustCompile(libgen.SearchMD5)
	var md5s []string
	for _, md5 := range md5Data {
		if re.MatchString(md5) {
			md5s = append(md5s, md5)
		}
	}

	return md5s, nil
}

// readInDOIs reads in a file of DOIs and returns an array of DOIs
//  for scientific magazines
func readInDOIs(fileName string) ([]string, error) {
	fileBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// remove brackets, quotes, and commas from the string file
	fileString := strings.ToLower(string(fileBytes))
	fileString = strings.Replace(fileString, "[", "", -1)
	fileString = strings.Replace(fileString, "]", "", -1)
	fileString = strings.Replace(fileString, "\"", "", -1)
	fileString = strings.Replace(fileString, ",", "", -1)
	fileString = strings.Replace(fileString, " ", "", -1)

	// split the string into an array of DOIs
	doiData := strings.Split(fileString, "\n")

	re := regexp.MustCompile(libgen.SearchDOI)
	var dois []string
	for _, doi := range doiData {
		if re.MatchString(doi) {
			dois = append(dois, doi)
		}
	}

	return dois, nil
}

const MaxFileSize = 1024 * 1024 * 1024 * 15 // 15MB

var downloadFromFileCmd = &cobra.Command{
	Use:     "download-from-file",
	Short:   "Download a specific set of resources by hash.",
	Long:    `Use this command if you already have the hashes of the specific resources you'd like to download in a file.'`,
	Example: "libgen download hashes.json",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) != 1 {
			if err := cmd.Help(); err != nil {
				fmt.Printf("error displaying CLI help: %v\n", err)
			}
			os.Exit(1)
		}

		// Get if magazine or book
		magazine, err := cmd.Flags().GetBool("magazine")
		if err != nil {
			fmt.Printf("error getting magazine flag: %v\n", err)
		}

		// Get flags
		output, err := cmd.Flags().GetString("output")
		if err != nil {
			fmt.Printf("error getting output flag: %v\n", err)
		}

		var hashes []string
		if magazine {
			// TODO: Finish adding in magazine downloads -- this is where it stops. This part doesn't work. --JA
			hashes, err = readInDOIs(args[0])
		} else {
			hashes, err = readInMd5s(args[0])
		}

		// Get the output file from the flags and get the hashes.
		if err != nil {
			fmt.Printf("error reading in hashes: %v\n", err)
			os.Exit(1)
		}

		// check if output is a directory and if not, make it one
		if output != "" {
			makeFolder(output)
		}

		if magazine {
			getMagazines(hashes, output)
			return
		}

		getBooks(hashes, output)
	},
}

func makeFolder(output string) {
	if _, err := os.Stat(output); os.IsNotExist(err) {
		fmt.Printf("Creating output directory: %s\n", output)
		if err := os.MkdirAll(output, 0755); err != nil {
			fmt.Printf("error making output directory: %v\n", err)
			os.Exit(1)
		}
	}
}

func ParseFilesize(filesize string) (int64, error) {
	var size int64
	var err error
	if strings.HasSuffix(filesize, "KB") {
		size, err = strconv.ParseInt(strings.TrimSuffix(filesize, "KB"), 10, 64)
		size *= 1024
	} else if strings.HasSuffix(filesize, "MB") {
		size, err = strconv.ParseInt(strings.TrimSuffix(filesize, "MB"), 10, 64)
		size *= 1024 * 1024
	} else if strings.HasSuffix(filesize, "GB") {
		size, err = strconv.ParseInt(strings.TrimSuffix(filesize, "GB"), 10, 64)
		size *= 1024 * 1024 * 1024
	} else {
		// check if it is only a number
		size, err = strconv.ParseInt(filesize, 10, 64)
		if err != nil {
			err = fmt.Errorf("unable to parse filesize: %s", filesize)
		}
	}
	return size, err
}

// getMagazines gets the magazines from the libgen API and writes them to a file
// TODO: Make this loop through, getting each GET Request and writing to a file. --JA
func getMagazines(dois []string, output string) {
	downloadMultipleMagazines(dois, output)
	return
}

// downloadMultipleMagazines downloads multiple magazines from the libgen API and writes them to a file
// TODO: Finish this up.
func downloadMultipleMagazines(dois []string, output string) {
	var wg sync.WaitGroup
	bChan := make(chan *libgen.ScienceMagazine, len(dois))
	fmt.Printf("%s %v MAGAZINES\n", color.GreenString("[DOWNLOADING] %v", len(dois)), color.YellowString("[PLEASE WAIT]"))
	for _, doi := range dois {
		download, err := libgen.GetScienceMagazineDownload(doi)
		if err != nil {
			fmt.Printf("error getting download: %v\n", err)
			continue
		}
		wg.Add(1)
		bChan <- &download
		go func() {
			if err := libgen.DownloadFile(<-bChan, output); err != nil {
				fmt.Printf("error downloading %v: %v\n", download.Title, err)
				os.Exit(1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	close(bChan)
}

func getBooks(md5s []string, output string) {

	books, err := libgen.GetDetails(&libgen.GetDetailsOptions{
		Hashes:       md5s,
		SearchMirror: libgen.GetWorkingMirror(libgen.SearchMirrors),
		Print:        true,
	})
	if err != nil {
		log.Fatalf("error retrieving results from LibGen API: %v", err)
	}

	// Check if the user wants to download all the books. If so, download them all.
	// Check if in CONST MaxFileSize, if so, download all the books.
	cleanedBooks := make([]*libgen.Book, 0)
	for _, book := range books {
		// parse the filesize string and convert to bytes
		filesize, err := ParseFilesize(book.Filesize)
		if err != nil {
			log.Fatalf("error parsing filesize: %v", err)
		}
		if filesize > MaxFileSize {
			fmt.Printf("%s: %s\n", color.RedString("[SKIPPED]"), book.Title)
			continue
		}
		cleanedBooks = append(cleanedBooks, book)
	}

	downloadMultipleBooks(cleanedBooks, output)

	if runtime.GOOS == "windows" {
		_, err = fmt.Fprintf(color.Output, "\n%s\n", color.GreenString("[DONE]"))
		if err != nil {
			fmt.Printf("error writing to Windows os.Stdout: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("\n%s\n", color.GreenString("[DONE]"))
	}
}

func downloadMultipleBooks(books []*libgen.Book, output string) {
	var wg sync.WaitGroup
	bChan := make(chan *libgen.Book, len(books))
	fmt.Printf("%s %v BOOKS\n", color.GreenString("[DOWNLOADING] %v", len(books)), color.YellowString("[PLEASE WAIT]"))
	for _, book := range books {
		if err := libgen.GetDownloadURL(book); err != nil {
			fmt.Printf("error getting download DownloadURL: %v\n", err)
			continue
		}
		wg.Add(1)
		bChan <- book
		go func() {
			if err := libgen.DownloadFile(<-bChan, output); err != nil {
				fmt.Printf("error downloading %v: %v\n", book.Title, err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	close(bChan)
}

type HashType uint8

const (
	MD5 HashType = iota
	DOI
	MIXED
	UNIDENTIFIABLE
)

func identifyFileContents(filepath string) HashType {
	file, err := os.Open(filepath)
	if err != nil {
		return UNIDENTIFIABLE
	}
	defer file.Close()

	// Read each line of the file and check if it is a DOI or MD5 hash through regexp.
	// If it is a DOI, return DOI. If it is a MD5 hash, return MD5.
	// If it is neither, return MIXED.
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if regexp.MustCompile(libgen.SearchMD5).MatchString(line) {
			return MD5
		}
		if regexp.MustCompile(libgen.SearchDOI).MatchString(line) {
			return DOI
		}
	}
	return UNIDENTIFIABLE
}

func init() {
	downloadFromFileCmd.Flags().StringP("output", "o", "", "where you want "+
		"libgen-cli to save your download.")
	downloadFromFileCmd.Flags().BoolP("magazine", "m", false, "are these magazines?")
}
