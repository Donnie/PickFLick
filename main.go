package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
)

// Movie struct
type Movie struct {
	Description string `json:"description"`
	Link        string `json:"link"`
	Poster      string `json:"poster"`
	Title       string `json:"title"`
}

func main() {
	// request and parse Kino.DE
	resp, err := http.Get("https://www.kino.de/filme/aktuell/?sp_country=deutschland")
	if err != nil {
		panic(err)
	}
	root, err := html.Parse(resp.Body)
	if err != nil {
		panic(err)
	}

	// Search for the title
	movies := []Movie{}

	lists := scrape.FindAll(root, scrape.ByClass("alice-teaser-media"))
	for _, list := range lists {
		movie := Movie{
			Description: scrape.Text(list.NextSibling.FirstChild.NextSibling),
			Link:        "https:" + scrape.Attr(list.NextSibling.FirstChild.FirstChild, "href"),
			Poster:      "https:" + scrape.Attr(list.FirstChild.FirstChild, "data-src"),
			Title:       scrape.Text(list.NextSibling.FirstChild.FirstChild),
		}
		movies = append(movies, movie)
	}

	b, _ := json.Marshal(movies)
	fmt.Println(string(b))
}
