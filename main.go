package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/codegangsta/cli"
	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
)

type entry struct {
	Title         string `json:"title"`
	URL           string `json:"url"`
	BookmarkCount int    `json:"bookmark_count"`
}

type QueryResult struct {
	Title   string
	Entries []*entry
}

type JsonFormat struct {
	Entries []*entry `json:"entries"`
	Error   string   `json:"error"`
}

func main() {
	app := cli.NewApp()
	app.Name = "gotebu"
	app.Usage = "Command Line Client for Hatena Bookmark HotEntry (http://b.hatena.ne.jp/hotentry)"
	app.ArgsUsage = "[category]"
	app.HideHelp = true
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "number, n",
			Value: 10,
			Usage: "number of output lines",
		},
		cli.BoolFlag{
			Name:  "json",
			Usage: "output as json",
		},
	}

	app.Action = func(c *cli.Context) {
		category := c.Args().First()
		number := c.Int("number")

		url := buildUrl(category)
		result, err := crawl(url, number)
		if c.Bool("json") {
			showResultAsJson(result, err)
		} else {
			showResult(result, url)
		}
	}

	app.Run(os.Args)
}

func buildUrl(keyword string) string {
	return fmt.Sprintf("http://b.hatena.ne.jp/hotentry/%s", url.QueryEscape(keyword))
}

func crawl(url string, number int) (QueryResult, error) {
	entries := []*entry{}
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return QueryResult{
			Title:   "",
			Entries: entries,
		}, err
	}
	doc.Find("div.top li").Each(func(_ int, line *goquery.Selection) {
		cells := [3]interface{}{"", "", 0}
		line.Find("h3").Each(func(_ int, s *goquery.Selection) {
			cells[0] = s.Text()
			s.Find("a").Each(func(_ int, s *goquery.Selection) {
				href, _ := s.Attr("href")
				if href != "" {
					cells[1] = href
				}
			})
		})
		count, _ := line.Attr("data-bookmark-count")
		if count != "" {
			if c, err := strconv.Atoi(count); err == nil {
				cells[2] = c
			}
		}

		entry := entry{
			Title:         strings.TrimSpace(cells[0].(string)),
			URL:           cells[1].(string),
			BookmarkCount: cells[2].(int),
		}
		if entry.URL != "" {
			entries = append(entries, &entry)
		}
	})

	if number > len(entries) {
		number = len(entries)
	}

	return QueryResult{
		Title:   doc.Find("title").Text(),
		Entries: entries[:number],
	}, nil
}

func maxTitleWidth(entries []*entry) int {
	width := 0
	for _, e := range entries {
		count := runewidth.StringWidth(e.Title)
		if count > width {
			width = count
		}
	}
	return width
}

func maxURLWidth(entries []*entry) int {
	width := 0
	for _, e := range entries {
		count := utf8.RuneCountInString(e.URL)
		if count > width {
			width = count
		}
	}
	return width
}

func showResult(result QueryResult, url string) {
	entries := result.Entries
	if len(entries) == 0 {
		fmt.Println("ホッテントリが見つかりませんでした ʕ◔ϖ◔ʔ")
		fmt.Printf("  url: %s\n\n", url)
		return
	}

	fmt.Printf("%s : %d 件\n",
		result.Title,
		len(entries),
	)
	fmt.Printf("  url: %s\n\n", url)

	titleWidth := maxTitleWidth(entries)
	titleFmt := fmt.Sprintf("%%-%ds", titleWidth)

	urlWidth := maxURLWidth(entries)
	urlFmt := fmt.Sprintf("%%-%ds", urlWidth)

	fmt.Fprintf(color.Output, " %s | %s | %s \n",
		color.BlueString(titleFmt, "Title"),
		fmt.Sprintf(urlFmt, "Url"),
		color.CyanString("Bookmark"),
	)
	fmt.Println(strings.Repeat("-", titleWidth+urlWidth+16))
	for _, e := range entries {
		fmt.Fprintf(color.Output, " %s | %s | %s \n",
			color.BlueString(runewidth.FillRight(e.Title, titleWidth)),
			fmt.Sprintf(urlFmt, e.URL),
			color.CyanString(strconv.Itoa(e.BookmarkCount)),
		)
	}
}

func showResultAsJson(result QueryResult, err error) {
	enc := json.NewEncoder(os.Stdout)
	if err != nil {
		enc.Encode(JsonFormat{Entries: []*entry{}, Error: err.Error()})
		return
	}
	err = enc.Encode(JsonFormat{Entries: result.Entries, Error: ""})
	if err != nil {
		fmt.Print(err)
	}
}
