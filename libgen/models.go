package libgen

type Book struct {
	Id        string
	Title     string
	Author    string
	Filesize  string
	Extension string
	Md5       string
	Year      string
	Url       string
}

const (
	SearchHref     = "<a href='book/index.php.+</a>"
	SearchMD5      = "[A-Z0-9]{32}"
	searchUrl      = "http://booksdl.org/get\\.php\\?md5=\\w{32}\\&key=\\w{16}"
	TitleMaxLength = 75
)

type BookFile struct {
	size int64
	name string
	path string
	data []byte
}
