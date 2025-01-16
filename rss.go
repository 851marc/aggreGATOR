package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func FetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	fmt.Println("Fetching feed ", feedURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "gator")

	fmt.Println("Executing request ", req)
	res, err := http.DefaultClient.Do(req)
	fmt.Println("Executed request ", res)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	fmt.Println("Fetched feed ", res.Body)

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var feed RSSFeed
	err = xml.Unmarshal(bytes, &feed)
	if err != nil {
		return nil, err
	}

	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	for i, v := range feed.Channel.Item {
		v.Title = html.UnescapeString(v.Title)
		v.Description = html.UnescapeString(v.Description)
		feed.Channel.Item[i] = v
	}

	return &feed, nil
}
