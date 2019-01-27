package libgen

import (
	"net/http"
	"net/url"
	"log"
	"regexp"
	"strings"
	"io/ioutil"
	"os"
)


const (
	SearchHref = "<a href='book/index.php.+</a>"
	SearchId = "book/index\\.php\\?md5=\\w{32}"
	SearchMD5 = "[A-Z0-9]{32}"
	SearchTitle = ">[^<]+"
	SearchUrl = "http://booksdl.org/get\\.php\\?md5=\\w{32}\\&key=\\w{16}"
	NumberOfBooks = "10"
)


type Book struct {
	title string
	hash string
}


func GetHref(HttpResponse string) (hrefs []string) {
	re := regexp.MustCompile(SearchUrl)
	matchs := re.FindAllString(HttpResponse, -1)

	for i := 0; i < len(matchs); i++ {
		log.Printf("%s\n", matchs[i])
	}

	return
}


func DownloadBooks(books []Book) {
	BaseUrl := &url.URL{
		Scheme: "http",
		Host: "booksdescr.org",
		Path: "ads.php",
	}

	for i := 0; i < len(books); i++ {
		q := BaseUrl.Query()
		q.Set("md5", books[i].hash)
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
					log.Printf(" ++ %s\n", books[i].title)
					GetHref(string(b))
				}
			}
		}
	}
	log.Println("......")
	log.Println("Finished")
}


func GetBooks(HttpResponse string) (books []Book) {
	re := regexp.MustCompile(SearchHref)
	matchs := re.FindAllString(HttpResponse, -1)

	for i := 0; i < len(matchs); i++ {
		re := regexp.MustCompile(SearchMD5)
		hash := re.FindString(matchs[i])
		if len(hash) != 0 {
			re := regexp.MustCompile(SearchTitle)
			title := re.FindString(matchs[i])
			title = strings.Replace(title, ">", "", 1)
			books = append(books, Book{title, hash})
		}
	}

	return
}

func RequestBooks(search string) (urls []string) {
	BaseUrl := &url.URL{
		Scheme: "http",
		Host: "libgen.io",
		Path: "search.php",
	}

	q := BaseUrl.Query()
	q.Set("req", search)
	q.Set("lg_topic", "libgen")
	q.Set("open", "0")
	q.Set("res", NumberOfBooks)
	q.Set("phrase", "1")
	q.Set("column", "def")
	BaseUrl.RawQuery = q.Encode()

	log.Printf("%s", BaseUrl.String())
	log.Println(".....")

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
	hashs := GetBooks(string(b))
	DownloadBooks(hashs)

	return
}
