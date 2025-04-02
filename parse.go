package main

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func parseLinks(html string) ([]string, error) {
	var links []string
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return links, err
	}

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			links = append(links, href)
		}
	})
	return links, nil
}
