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

package libgen_cli

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/ciehanski/libgen-cli/libgen"
)

type BookSelection []*libgen.Book

func (b *BookSelection) RemoveBook(remove *libgen.Book) {
	for i, bk := range *b {
		if bk.Md5 == remove.Md5 {
			slice := append((*b)[:i], (*b)[i+1:]...)
			*b = slice
			break
		}
	}
}

func (b *BookSelection) AddBook(book *libgen.Book) {
	*b = append(*b, book)
}

type BookCliSelections []string
type BookCliInterface interface {
	SetSelected(book *libgen.Book)
	SetUnselected(book *libgen.Book)
	GetStrings() []string
}

func getBookInterface(books []string) (a BookCliInterface) {
	return (*BookCliSelections)(&books)
}

// UpdateBookCount updates the book count in the cli prompt
func (b *BookCliSelections) UpdateBookCount(addedBooks []*libgen.Book) {
	// if len(findBookCountInString((*b)[len(*b)-1])) != 0 {
	// 	(*b)[len(*b)-1] = strings.Replace((*b)[len(*b)-1], findBookCountInString((*b)[len(*b)-1]), fmt.Sprintf("%v books selected", newly), -1)
	// } else {
	(*b)[len(*b)-1] = "Finish"

	// }
}

// findBookCountInString
func findBookCountInString(str string) string {
	regex := regexp.MustCompile(`\b\d+ books selected\b`)
	return regex.FindString(str)
}

func (b *BookCliSelections) ToggleSelected(book *libgen.Book) bool {
	if book.ID == "-1" {
		return false
	}
	if book.Selected {
		b.SetUnselected(book)
	} else {
		b.SetSelected(book)
	}
	return true
}

// SetSelected changes the cli prompt to show the book as selected (✔)
func (b *BookCliSelections) SetSelected(book *libgen.Book) {
	for i, bk := range *b {
		if strings.Contains(bk, book.Md5[len(book.Md5)-8:]) == true {
			(*b)[i] = strings.Replace(bk, book.Md5[len(book.Md5)-8:], fmt.Sprintf("✔ %s", book.Md5[len(book.Md5)-8:]), -1)
			book.Selected = true
			return
		}
	}
}

// SetUnselected changes the cli prompt to REMOVE the book as selected: removes the (✔)
func (b *BookCliSelections) SetUnselected(book *libgen.Book) {
	for i, bk := range *b {
		if strings.Contains(bk, book.Md5[len(book.Md5)-8:]) == true {
			(*b)[i] = strings.Replace(bk, fmt.Sprintf("✔ %s", book.Md5[len(book.Md5)-8:]), fmt.Sprintf("%s", book.Md5[len(book.Md5)-8:]), -1)
			book.Selected = false
			return
		}
	}
}

func (b *BookCliSelections) AddBook(book *libgen.Book) {
	*b = append(*b, formatBookCli(book))
	fmt.Println("Added book:", book.Title)
}

func (b *BookCliSelections) GetStrings() []string {
	var strings []string
	for _, book := range *b {
		strings = append(strings, book)
	}
	return strings
}

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:     "search",
	Short:   "Query all content hosted by Library Genesis.",
	Long:    `Searches for all resources that result from the provided query and then provides them for download.`,
	Example: "libgen search kubernetes",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 1 {
			if err := cmd.Help(); err != nil {
				fmt.Printf("error displaying CLI help: %v\n", err)
			}
			os.Exit(1)
		}

		// Get flags
		results, err := cmd.Flags().GetInt("results")
		if err != nil {
			fmt.Printf("error getting results flag: %v\n", err)
		}
		requireAuthor, err := cmd.Flags().GetBool("require-author")
		if err != nil {
			fmt.Printf("error getting require-author flag: %v\n", err)
		}
		extension, err := cmd.Flags().GetStringSlice("extension")
		if err != nil {
			fmt.Printf("error getting extension flag: %v\n", err)
		}
		output, err := cmd.Flags().GetString("output")
		if err != nil {
			fmt.Printf("error getting output flag: %v\n", err)
		}
		year, err := cmd.Flags().GetInt("year")
		if err != nil {
			fmt.Printf("error getting output flag: %v\n", err)
		}
		publisher, err := cmd.Flags().GetString("publisher")
		if err != nil {
			fmt.Printf("error getting publisher flag: %v\n", err)
		}
		language, err := cmd.Flags().GetString("language")
		if err != nil {
			fmt.Printf("error getting language flag: %v\n", err)
		}

		// Join args for complete search query in case
		// it contains spaces
		searchQuery := strings.Join(args, " ")
		fmt.Printf("++ Searching for: %s\n", searchQuery)

		var books []*libgen.Book
		var searchMirror = libgen.GetWorkingMirror(libgen.SearchMirrors)
		books, err = libgen.Search(&libgen.SearchOptions{
			Query:         searchQuery,
			SearchMirror:  searchMirror,
			Results:       results,
			Print:         true,
			RequireAuthor: requireAuthor,
			Extension:     extension,
			Year:          year,
			Publisher:     publisher,
			Language:      language,
		})
		if err != nil {
			fmt.Printf("error completing search query: %v\n", err)
			os.Exit(1)
		}
		if len(books) == 0 {
			fmt.Printf("\nNo results found from: %s.\n", searchMirror.String())
			os.Exit(1)
		}

		books = append(books, &libgen.Book{
			ID:          "-1",
			Title:       "Downloads",
			Md5:         "Exit Now",
			Author:      "",
			Filesize:    "0",
			Extension:   "0 books selected",
			Year:        "2022",
			DownloadURL: "Download.com",
			Selected:    false,
		})

		var bookSelection []string
		for _, b := range books {
			bookSelection = append(bookSelection, formatBookCli(b))
		}

		// TODO: Add support for multiple selections
		promptTemplate := &promptui.SelectTemplates{
			Active: `▸ {{ .Md5 | cyan | bold }}{{ if .Title }} ({{ .Title }}){{end}}`,
			// Inactive: `  {{ .Title | cyan }}{{ if .Title }} ({{ .Title }}){{end}}`,
			Selected: `{{ "✔" | green }} %s: {{ .Md5 | cyan }}{{ if .Title }} ({{ .Title }}){{end}}`,
		}

		bookSelectionFunc := func(bookSelection *[]string) []string {
			return *bookSelection
		}(&bookSelection)

		prompt := promptui.Select{
			Label: "Select Book",
			// Items: bookSelection,
			Items:     bookSelectionFunc,
			Templates: promptTemplate,
			Size:      results,
			IsVimMode: false,
			Keys: &promptui.SelectKeys{
				Next: promptui.Key{
					Code:    readline.CharNext,
					Display: "↓ (j)",
				},
				Prev: promptui.Key{
					Code:    readline.CharPrev,
					Display: "↑ (k)",
				},
				PageUp: promptui.Key{
					Code:    readline.CharForward,
					Display: "→ (l)",
				},
				PageDown: promptui.Key{
					Code:    readline.CharBackward,
					Display: "← (h)",
				},
			},
		}

		fmt.Println(strings.Repeat("-", 80))

		loopdeloop := 0

		var selectBooks = make([]*libgen.Book, 0)
		for loopdeloop < 10 {
			loopdeloop++
			_, result, err := prompt.Run()
			if err != nil {
				fmt.Print(err)
				os.Exit(1)
			}

			var selectedBook libgen.Book
			for i, b := range bookSelection {
				if b == result {
					selectedBook = *books[i]

					// make sure we don't download the same book twice. pass in the origin book from the array.
					ok := (*BookCliSelections)(&bookSelection).ToggleSelected(books[i])
					if !ok {
						goto Download
					}
					(*BookCliSelections)(&bookSelection).UpdateBookCount(selectBooks)

					if !selectedBook.Selected {
						(*BookSelection)(&selectBooks).AddBook(&selectedBook)
						break
					}

					// remove &selectedBook from selectedBooks
					(*BookSelection)(&selectBooks).RemoveBook(&selectedBook)
					break
				}
			}
		}

	Download:
		download(selectBooks, DownloadConfig{
			DownloadConstraints: 3,
			Output:              output,
			DownloadType:        ConcurrencyWithConstraints,
		})

		return
		if len(selectBooks) > 1 {
			downloadMultipleBooks(selectBooks, output)
			return
		}
		selectedBook := selectBooks[0]
		if selectedBook.Author == "" {
			fmt.Printf("Download starting for: %s by N/A\n", selectedBook.Title)
		} else {
			fmt.Printf("Download starting for: %s by %s\n", selectedBook.Title, selectedBook.Author)
		}

		if err := libgen.GetDownloadURL(selectedBook); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := libgen.DownloadFile(selectedBook, output); err != nil {
			fmt.Printf("error downloading %v: %v\n", selectedBook.Title, err)
			os.Exit(1)
		}

		if runtime.GOOS == "windows" {
			_, err = fmt.Fprintf(color.Output, "\n%s %s by %s.%s", color.GreenString("[OK]"),
				selectedBook.Title, selectedBook.Author, selectedBook.Extension)
			if err != nil {
				fmt.Printf("error writing to Windows os.Stdout: %v\n", err)
			}
		} else {
			fmt.Printf("\n%s %s by %s.%s\n", color.GreenString("[OK]"),
				selectedBook.Title, selectedBook.Author, selectedBook.Extension)
		}
	},
}

// formatBookCli formats a book for CLI output
//  @param b *libgen.Book - the book to format for CLI output
//  @return string
func formatBookCli(b *libgen.Book) string {
	var pBookFormat string

	shortMd5 := b.Md5[len(b.Md5)-8:]
	selectChoice := fmt.Sprintf("%8s ", color.New(color.FgHiBlue).Sprintf(shortMd5))
	if len(b.Title) > 36 {
		pBookFormat = b.Title[:36] + "... by"
	} else {
		pBookFormat = b.Title + " by"
	}
	selectChoice += fmt.Sprintf("%s ", pBookFormat)
	if b.Author != "" {
		if len(b.Author) > 20 {
			selectChoice += fmt.Sprintf("%s ", color.New(color.FgYellow).Sprintf(b.Author[:17]+"..."))
		} else {
			selectChoice += fmt.Sprintf("%s ", color.New(color.FgYellow).Sprintf(b.Author))
		}
	} else {
		selectChoice += fmt.Sprintf("%s ", color.New(color.FgYellow).Sprintf("N/A"))
	}
	selectChoice += fmt.Sprintf("| %-4s ", color.New(color.FgRed).Sprintf(b.Extension))
	size, err := strconv.Atoi(b.Filesize)
	if err != nil {
		fmt.Printf("error converting string to int: %v\n", err)
		os.Exit(1)
	}
	selectChoice += fmt.Sprintf("| %v", color.New(color.FgGreen).Sprintf(humanize.Bytes(uint64(size))))
	return selectChoice
}

func init() {
	searchCmd.Flags().IntP("results", "r", 10, "controls how many "+
		"query results are displayed.")
	searchCmd.Flags().BoolP("require-author", "a", false, "controls "+
		"if the query results will return any media without a listed author.")
	searchCmd.Flags().StringSliceP("extension", "e", []string{""}, "controls if the query "+
		"results will return any media with a certain file extension.")
	searchCmd.Flags().StringP("output", "o", "", "where you want "+
		"libgen-cli to save your download.")
	searchCmd.Flags().IntP("year", "y", 0, "filters search query results by the "+
		"year provided.")
	searchCmd.Flags().StringP("publisher", "p", "", "filters search query "+
		"results by the publisher provided")
	searchCmd.Flags().StringP("language", "l", "", "filters search query "+
		"results by the language provided")
}
