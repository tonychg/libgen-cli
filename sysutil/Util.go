package sysutil

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func MakeFolder(output string) {
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

const DoiFile = "do-files.txt"
const MagazineCache = "magazine-cache"

func CheckIfDoiIsInFileList(doi string) bool {
	file, err := os.Open(DoiFile)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == doi {
			return true
		}
	}
	return false
}

func SaveDoiToFileList(doi string) {
	// if the file doesn't exist, create it
	if _, err := os.Stat(DoiFile); os.IsNotExist(err) {
		file, err := os.Create(DoiFile)
		if err != nil {
			fmt.Printf("error creating file: %v\n", err)
			return
		}
		defer file.Close()
	}
	f, err := os.OpenFile(DoiFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	// write the doi to the file followed by a newline
	if _, err = f.WriteString(doi + "\n"); err != nil {
		log.Println(err)
	}
}
