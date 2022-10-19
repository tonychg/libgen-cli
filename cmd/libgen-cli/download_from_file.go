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
	"encoding/json"
	"fmt"
	"github.com/ciehanski/libgen-cli/sysutil"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

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
	fileString = strings.Replace(fileString, "\\", "", -1)
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
			downloadSciMags(hashes, DownloadConfig{
				DelaySeconds:        2,
				DownloadConstraints: 1,
				Output:              output,
				DownloadType:        NoConcurrency,
			})
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
		filesize, err := sysutil.ParseFilesize(book.Filesize)
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

type DownloadType int

const (
	ConcurrencyWithConstraints DownloadType = iota
	Concurrency
	BatchConcurrency
	NoConcurrency
)

type DownloadConfig struct {
	FetchDelaySeconds   int
	DelaySeconds        int
	DownloadConstraints int
	Output              string
	DownloadType        DownloadType
}

// download gets selects the correct function to download the
//  books based on the config.
func download(books []*libgen.Book, config DownloadConfig) {
	if len(books) == 1 {
		downloadSingleBook(books[0], config)
		return
	}

	switch config.DownloadType {
	case ConcurrencyWithConstraints:
		downloadConcurrentWithConstraints(books, config)
		break
	case Concurrency:
		downloadMultipleBooks(books, config.Output)
		break
	case BatchConcurrency:
		batchDownloadBooks(books, config)
		break
	case NoConcurrency:
		for _, book := range books {
			downloadSingleBook(book, config)
		}
		break
	}
}

// downloadSingleBook downloads a single book from the libgen API and writes it to a file
func downloadSingleBook(book *libgen.Book, config DownloadConfig) {
	fmt.Printf("%s %v Single\n", color.GreenString("[DOWNLOADING]"), color.YellowString("[PLEASE WAIT]"))
	fmt.Printf("Download starting for: %s by %s\n", book.Title, book.Author)
	if err := libgen.GetDownloadURL(book); err != nil {
		fmt.Printf("error getting download DownloadURL: %v\n", err)
		return
	}
	if err := libgen.DownloadFile(book, config.Output); err != nil {
		fmt.Printf("error downloading %v: %v\n", book.Title, err)
	}
	if config.DelaySeconds > 0 {
		time.Sleep(time.Duration(config.DelaySeconds) * time.Second)
	}
}

// downloadMultipleBooks downloads multiple books from the libgen API and writes them to a file
// using concurrency
func downloadMultipleBooks(books []*libgen.Book, output string, config ...DownloadConfig) {
	fmt.Printf("%s %v Concurrently\n", color.GreenString("[DOWNLOADING] %v", len(books)), color.YellowString("[PLEASE WAIT]"))

	var wg sync.WaitGroup

	dlLimit := len(books)
	if config != nil && config[0].DownloadConstraints > 0 {
		dlLimit = config[0].DownloadConstraints
	}
	bChan := make(chan *libgen.Book, dlLimit)

	fmt.Printf("%s %v BOOKS\n", color.GreenString("[DOWNLOADING] %v", len(books)), color.YellowString("[PLEASE WAIT]"))
	for _, book := range books {
		if err := libgen.GetDownloadURL(book); err != nil {
			fmt.Printf("error getting download DownloadURL: %v\n", err)
			continue
		}
		time.Sleep(time.Millisecond * 1000)
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

// batchDownloadBooks downloads multiple books from the libgen API and writes them to a file
//  using batching. This is the default method of downloading books. It means that only a limited
//  number of books can be downloaded at a time, per batch. So if the batch max capacity is x,
//  then 0 < a < x books must be downloaded before the next batch can be started/downloaded.
func batchDownloadBooks(books []*libgen.Book, config DownloadConfig) {
	fmt.Printf("%s %v %s\n", color.GreenString("[DOWNLOADING] %v", len(books)), color.YellowString("[PLEASE WAIT]"), color.GreenString("[BATCH DOWNLOAD]"))

	maxBatch := config.DownloadConstraints
	bChans := make([]chan struct{}, maxBatch)
	for i := 0; i < maxBatch; i++ {
		bChans[i] = make(chan struct{}, 1)
	}
	bookCount := 0
	breakPoint := -1
	for {
		for i := 0; i < maxBatch; i++ {
			if bookCount < len(books) {
				go func(b chan struct{}, book *libgen.Book) {
					downloadSingleBook(book, config)
					b <- struct{}{}
				}(bChans[i], books[bookCount])
			} else {
				breakPoint = i
				break
			}
			bookCount++
		}

		for i := 0; i < maxBatch; i++ {
			if i == breakPoint {
				break
			}
			if i != -1 || i < breakPoint {
				select {
				case <-bChans[i]:
					fmt.Printf("%d\n", i)
					continue
				}
			}
		}
		if breakPoint != -1 {
			break
		}
	}
}

// downloadConcurrentWithConstraints downloads multiple books from the libgen API
//  and writes them to a file. This method will always be downloading x books at a time.
//  If the number of books is less than x, then the remaining books will be downloaded.
//  If the number of books is greater than x, we loop in batches of x, downloading x at a time.
//
// Example:
//  If the number of books is 7, and we set the max downloads at 3, then we will download 3 books at a time.
//  BookA (3Mb), BookB (5Mb), BookC (2Mb), BookD (4Mb), BookE (8Mb), BookF (1Mb), BookG (15Mb)
//  It starts downloading Book1 (5Mb), Book2 (2Mb), Book3 (8Mb).
//  BookC will finish downloading first. Then BookD will start downloading.
//   - BookA (1Mb Left), BookB (3Mb Left), BookD (4Mb)
//  BookA will finish downloading next. Then BookE will start downloading.
//   - BookE (8Mb), BookB (2Mb Left), BookD (3Mb)
//  BookB will finish downloading next. Then BookF will start downloading.
//   - BookE (6Mb Left), BookF (1Mb), BookD (1Mb Left)
//  BookD and BookF will finish downloading next. Then BookG will start downloading.
//   - BookE (5Mb Left), BookG (15Mb)
//   - BookG (9Mb Left)
//  Finishes Downloading All Books.
func downloadConcurrentWithConstraints(books []*libgen.Book, config DownloadConfig) {
	fmt.Printf("%s %v Batch\n", color.GreenString("[DOWNLOADING WITH MAX] %v", len(books)), color.YellowString("[PLEASE WAIT]"))

	// always be downloading a max of x books at a time. no more, no less.
	sChan := make(chan struct{}, config.DownloadConstraints)
	wgBooks := sync.WaitGroup{}
	wgBooks.Add(len(books))

	for i, book := range books {
		sChan <- struct{}{}
		fmt.Printf("%s\n", color.HiYellowString("Download %v Started...", i))
		go func(book *libgen.Book, s chan struct{}, wg *sync.WaitGroup) {
			downloadSingleBook(book, config)
			<-sChan
			wgBooks.Done()
		}(book, sChan, &wgBooks)
	}

	wgBooks.Wait()

	maxBatch := config.DownloadConstraints
	bChans := make([]chan struct{}, maxBatch)
	for i := 0; i < maxBatch; i++ {
		bChans[i] = make(chan struct{}, 1)
	}
}

type HashType uint8

const (
	MD5 HashType = iota
	DOI
	MIXED
	UNIDENTIFIABLE
)

// identifyFileContents looks within the file and decides whether it is md5 or DOI.
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

	isMd5 := false
	isDoi := false

	// Check if the file is a DOI or MD5 hash.
	for scanner.Scan() {
		line := scanner.Text()
		if regexp.MustCompile(libgen.SearchMD5).MatchString(line) {
			isMd5 = true
		}
		if regexp.MustCompile(libgen.SearchDOI).MatchString(line) {
			isDoi = true
		}
	}

	// If it is a DOI, return DOI. If it is a MD5 hash, return MD5. If it has both, or neither, return UNIDENTIFIABLE
	if isDoi && !isMd5 {
		return DOI
	}
	if isMd5 && !isDoi {
		return MD5
	}

	return UNIDENTIFIABLE
}

// downloadMultipleMagazines downloads multiple magazines from the libgen API and writes them to a file
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

func fetchMagazineDownloadUrl(doi string) *libgen.ScienceMagazine {
	var download libgen.ScienceMagazine

	// check if the file with the DOI exists in the file cache. if so, load it and unmarshal it into the download struct.
	// if not, then fetch the download url from the libgen API and marshal it into the file cache.
	// replace all but alphanumeric from the DOI

	filename := regexp.MustCompile("[^a-zA-Z0-9]+").ReplaceAllString(doi, "")
	if _, err := os.Stat(fmt.Sprintf("%s/%s.json", sysutil.MagazineCache, filename)); err == nil {
		file, err := os.Open(fmt.Sprintf("%s/%s.json", sysutil.MagazineCache, filename))
		if err != nil {
			fmt.Printf("error opening file: %v\n", err)
			return nil
		}
		defer file.Close()

		if err := json.NewDecoder(file).Decode(&download); err != nil {
			fmt.Printf("error decoding file: %v\n", err)
			return nil
		}
		return &download
	}

	var err error
	if download, err = libgen.GetScienceMagazineDownload(doi); err != nil {
		fmt.Printf("error getting source: %v\n", err)
	}
	fmt.Printf("%s\n", color.GreenString("[DOWNLOAD URL FETCHED] %v", download.Title))

	file, err := os.Create(fmt.Sprintf("%s/%s.json", sysutil.MagazineCache, filename))
	if err != nil {
		fmt.Printf("error creating file: %v\n", err)
		return nil
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(download); err != nil {
		fmt.Printf("error encoding file: %v\n", err)
		return nil
	}

	return &download
}

// downloadSingleBook downloads a single book from the libgen API and writes it to a file
func downloadSingleMagazine(download *libgen.ScienceMagazine, config DownloadConfig) {
	fmt.Printf("%s %v Single\n", color.GreenString("[DOWNLOADING]"), color.YellowString("[PLEASE WAIT]"))
	if err := libgen.DownloadFile(download, config.Output); err != nil {
		fmt.Printf("error downloading %v: %v\n", download.Title, err)
		return
	}
	// save the doi to the file list
	sysutil.SaveDoiToFileList(download.DOI)

	if config.DelaySeconds > 0 {
		time.Sleep(time.Duration(config.DelaySeconds) * time.Second)
	}
}

func fetchMagazineDownloadUrls(dois []string, config DownloadConfig) []*libgen.ScienceMagazine {
	// check if the doi is in the file list for each doi. if so, remove the doi from the list
	newDoiList := []string{}
	for i, doi := range dois {
		if sysutil.CheckIfDoiIsInFileList(doi) {
			fmt.Printf("%s %v\n", color.HiYellowString("Skipping %v...", i), color.HiYellowString(doi))
			continue
		}
		newDoiList = append(newDoiList, dois[i])
	}
	dois = newDoiList
	fmt.Printf("%s %v MAGAZINES\n", color.GreenString("[FETCHING SOURCES] %v", len(dois)), color.YellowString("[PLEASE WAIT]"))
	cmChan := make(chan *libgen.ScienceMagazine, len(dois))
	defer close(cmChan)
	sChan := make(chan struct{}, config.DownloadConstraints)
	defer close(sChan)

	var wg sync.WaitGroup
	wg.Add(len(dois))
	for _, doi := range dois {
		fmt.Printf("%s %v\n", color.GreenString("[FETCH STARTED] %v", doi), color.YellowString("[PLEASE WAIT]"))
		sChan <- struct{}{}

		// fetch the download url
		go func(doi string, s chan struct{}, wg *sync.WaitGroup, cm chan *libgen.ScienceMagazine) {
			cmChan <- fetchMagazineDownloadUrl(doi)
			if config.FetchDelaySeconds > 0 {
				time.Sleep(time.Duration(config.DelaySeconds) * time.Second)
			}
			<-sChan
			wg.Done()
		}(doi, sChan, &wg, cmChan)
	}

	fmt.Printf("%s %v FINISHING UP\n", color.GreenString("[FETCHING SOURCES] %v", len(dois)), color.YellowString("[PLEASE WAIT]"))
	wg.Wait()
	var downloads []*libgen.ScienceMagazine
	fmt.Printf("%s %v FINISHED\n", color.GreenString("[FETCHING SOURCES] %v", len(dois)), color.YellowString("[PLEASE WAIT]"))

	for i := 0; i < len(dois); i++ {
		url := <-cmChan
		if url != nil {
			downloads = append(downloads, url)
		}
	}
	return downloads
}

func downloadMagazinesConcurrentWithConstraints(books []*libgen.ScienceMagazine, config DownloadConfig) {
	fmt.Printf("%s %v\n", color.GreenString("[DOWNLOADING WITH MAX] %v", len(books)), color.YellowString("[PLEASE WAIT]"))

	// always be downloading a max of x books at a time. no more, no less.
	sChan := make(chan struct{}, config.DownloadConstraints)
	wgBooks := sync.WaitGroup{}
	wgBooks.Add(len(books))

	for i, book := range books {
		sChan <- struct{}{}
		fmt.Printf("%s\n", color.HiYellowString("Download %v Started...", i))
		go func(book *libgen.ScienceMagazine, s chan struct{}, wg *sync.WaitGroup) {
			downloadSingleMagazine(book, config)
			<-sChan
			wgBooks.Done()
		}(book, sChan, &wgBooks)
	}

	wgBooks.Wait()

	maxBatch := config.DownloadConstraints
	bChans := make([]chan struct{}, maxBatch)
	for i := 0; i < maxBatch; i++ {
		bChans[i] = make(chan struct{}, 1)
	}
}

func downloadSciMags(dois []string, config DownloadConfig) {
	downloads := fetchMagazineDownloadUrls(dois, config)

	if config.DownloadType == ConcurrencyWithConstraints {
		downloadMagazinesConcurrentWithConstraints(downloads, config)
	} else {
		for i, magazine := range downloads {
			fmt.Printf("%s %v\n", color.GreenString("[DOWNLOADING STARTED] %v", i), color.YellowString("[PLEASE WAIT]"))
			downloadSingleMagazine(magazine, config)
		}
	}
}

func init() {
	downloadFromFileCmd.Flags().StringP("output", "o", "", "where you want "+
		"libgen-cli to save your download.")
	downloadFromFileCmd.Flags().BoolP("magazine", "m", false, "are these magazines?")
}
