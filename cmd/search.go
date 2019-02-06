// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
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

package cmd

import (
	"os"
	"log"
	"fmt"
	"strings"

	"github.com/tonychg/libgen-cli/libgen"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

const TitleMaxLength = 60

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search some pattern on libgen.io",
	Long: `Search pattern and get a list of hash map urls to it, and show
 formated title + link`,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			books []libgen.Book
			books_title []string
			subtitle string
		)

		if len(args) < 1 {
			log.Fatal("Error: Search need a pattern for the command")
			os.Exit(1)
		}

		pattern := strings.Join(args, " ")
		log.Printf(" ++ Searching: %s\n", pattern)

		hashes := libgen.Search(pattern, 10)
		books = libgen.GetDetails(hashes)

		for _, b := range books {
			selectChoice := fmt.Sprintf("%8s ", b.Id)
			selectChoice += fmt.Sprintf("%-4s ", b.Extension)
			if len(b.Title) > TitleMaxLength {
				subtitle = b.Title[:TitleMaxLength]
			} else {
				subtitle = b.Title
			}
			selectChoice += fmt.Sprintf("%s", subtitle)
			books_title = append(books_title, selectChoice)
		}

		prompt := promptui.Select{
			Label: "Select Book",
			Items: books_title,
		}

		_, result, err := prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		var selected libgen.Book

		for i, b := range books_title {
			if b == result {
				selected = books[i]
				break
			}
		}

		libgen.GetDownloadUrl(&selected)
		libgen.DownloadBook(selected)
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// searchCmd.PersistentFlags().String("BookName", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// searchCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
