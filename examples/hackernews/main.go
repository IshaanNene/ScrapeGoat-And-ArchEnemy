// Hacker News Scraper — scrapes front page stories
// Extracts: rank, title, URL, points, author
package main

import (
	"fmt"
	"strings"
	"time"

	scrapegoat "github.com/IshaanNene/ScrapeGoat/pkg/scrapegoat"
)

func main() {
	fmt.Println("   Hacker News Scraper")
	fmt.Println("   Extracting top stories: rank, title, URL, points, author")
	fmt.Println("   Starting crawl...")

	crawler := scrapegoat.NewCrawler(
		scrapegoat.WithConcurrency(3),
		scrapegoat.WithMaxDepth(1),
		scrapegoat.WithDelay(500*time.Millisecond),
		scrapegoat.WithOutput("json", "./output/hackernews"),
		scrapegoat.WithAllowedDomains("news.ycombinator.com"),
		scrapegoat.WithMaxRequests(10),
	)

	// Follow ONLY the "More" pagination link — skip vote/item/login/submit links
	crawler.OnHTML("a.morelink[href]", func(e *scrapegoat.Element) {
		e.Follow(e.Attr("href"))
	})

	// Extract each story row
	crawler.OnHTML("tr.athing", func(e *scrapegoat.Element) {
		rank := strings.TrimSuffix(strings.TrimSpace(e.Selection.Find(".rank").Text()), ".")
		title := e.Selection.Find(".titleline > a").First().Text()
		href, _ := e.Selection.Find(".titleline > a").First().Attr("href")
		site := strings.TrimSpace(e.Selection.Find(".sitestr").Text())

		// Metadata from sibling row
		subRow := e.Selection.Next()
		points := strings.TrimSpace(subRow.Find(".score").Text())
		author := strings.TrimSpace(subRow.Find(".hnuser").Text())
		commentLink := subRow.Find("a").Last().Text()

		if title != "" {
			e.Item.Set("rank", rank)
			e.Item.Set("title", title)
			e.Item.Set("url", href)
			e.Item.Set("site", site)
			e.Item.Set("points", points)
			e.Item.Set("author", author)
			e.Item.Set("comments", strings.TrimSpace(commentLink))
		}
	})

	if err := crawler.Start("https://news.ycombinator.com"); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	crawler.Wait()

	stats := crawler.Stats()
	fmt.Printf("\n✅ Complete! Stories: %v | Pages: %v\n",
		stats["items_scraped"], stats["requests_sent"])
}
