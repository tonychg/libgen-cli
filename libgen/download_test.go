package libgen

import (
	"testing"
)

func TestDownloadBook(t *testing.T) {
	book, err := GetDetails([]string{"2F2DBA2A621B693BB95601C16ED680F8"}, false, false, "")
	if err != nil {
		t.Error(err)
	}

	if err := DownloadBook(book[0], ""); err != nil {
		t.Error(err)
	}
}

func TestGetDownloadURL(t *testing.T) {
	book, err := GetDetails([]string{"2F2DBA2A621B693BB95601C16ED680F8"}, false, false, "")
	if err != nil {
		t.Error(err)
	}

	if err := getDownloadURL(&book[0]); err != nil {
		t.Error(err)
	}

	if book[0].URL == "" {
		t.Error()
	}
}

func TestGetBokDownloadURL(t *testing.T) {
	book, err := GetDetails([]string{"2F2DBA2A621B693BB95601C16ED680F8"}, false, false, "")
	if err != nil {
		t.Error(err)
	}

	if err := getBokDownloadURL(&book[0]); err != nil {
		t.Error(err)
	}
}

func TestGetBooksdlDownloadURL(t *testing.T) {
	book, err := GetDetails([]string{"2F2DBA2A621B693BB95601C16ED680F8"}, false, false, "")
	if err != nil {
		t.Error(err)
	}

	if err := getBooksdlDownloadURL(&book[0]); err != nil {
		t.Error(err)
	}
}
