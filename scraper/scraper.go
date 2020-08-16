package scraper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
)

// Movie struct
type Movie struct {
	Description string `json:"description"`
	Link        string `json:"link"`
	Poster      string `json:"poster"`
	Rank        int    `json:"rank"`
	Title       string `json:"title"`
}

// Save scrapes the website and saves data locally as json to a file
func Save(file string) {
	fmt.Println("Scraping started at", time.Now())
	var startPage = "https://www.kino.de/filme/aktuell/?sp_country=deutschland"
	// request and parse Kino.DE
	resp, err := http.Get(startPage)
	if err != nil {
		panic(err)
	}
	root, err := html.Parse(resp.Body)
	if err != nil {
		panic(err)
	}

	// Get all pages
	pagination, _ := scrape.Find(root, scrape.ByClass("alice-pagination-default"))
	pages := strings.Fields(scrape.Text(pagination))
	pages = pages[1 : len(pages)-1]

	for i, page := range pages {
		if page == "1" {
			pages[i] = startPage
		} else {
			pages[i] = "https://www.kino.de/filme/aktuell/page/" + page + "/?sp_country=deutschland"
		}
	}

	movies := []Movie{}
	rank := 0

	for _, link := range pages {
		movies = append(movies, getPage(link, &rank)...)
		time.Sleep(2 * time.Second)
	}

	data, _ := json.MarshalIndent(movies, "", " ")
	ioutil.WriteFile(file, data, 0644)
	fmt.Println("Scraping ended at", time.Now())
}

func getPage(link string, rank *int) (movies []Movie) {
	resp, err := http.Get(link)
	if err != nil {
		panic(err)
	}
	root, err := html.Parse(resp.Body)
	if err != nil {
		panic(err)
	}

	lists := scrape.FindAll(root, scrape.ByClass("alice-teaser-media"))
	for _, list := range lists {
		*rank++
		posterLink := strings.Replace(scrape.Attr(list.FirstChild.FirstChild, "data-src"), "160x226u", "320x452u", -1)
		movie := Movie{
			Description: scrape.Text(list.NextSibling.FirstChild.NextSibling),
			Link:        "https:" + scrape.Attr(list.NextSibling.FirstChild.FirstChild, "href"),
			Poster:      "https:" + posterLink,
			Rank:        *rank,
			Title:       scrape.Text(list.NextSibling.FirstChild.FirstChild),
		}
		movies = append(movies, movie)
	}

	return
}
