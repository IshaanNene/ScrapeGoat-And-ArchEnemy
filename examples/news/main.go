// Multi-Site News Aggregator — crawls multiple news sources
// Extracts structured data (JSON-LD, OpenGraph, meta) from any news site
package main

import (
	"fmt"
	"time"

	scrapegoat "github.com/IshaanNene/ScrapeGoat/pkg/scrapegoat"
)

func main() {
	fmt.Println("📰 Multi-Site News Aggregator")
	fmt.Println("   Auto-extracting structured data from multiple news sites")

	crawler := scrapegoat.NewCrawler(
		scrapegoat.WithConcurrency(8),
		scrapegoat.WithMaxDepth(1),
		scrapegoat.WithDelay(300*time.Millisecond),
		scrapegoat.WithOutput("jsonl", "./output/news"),
		scrapegoat.WithMaxRequests(100),
		scrapegoat.WithRobotsRespect(true),
	)

	// Follow article links
	crawler.OnHTML("a[href]", func(e *scrapegoat.Element) {
		href := e.Attr("href")
		// Only follow links that look like articles
		if len(href) > 20 {
			e.Follow(href)
		}
	})

	// Extract headline and content preview
	crawler.OnHTML("article, .article, .post", func(e *scrapegoat.Element) {
		title := e.Selection.Find("h1, h2").First().Text()
		body := e.Selection.Find("p").First().Text()
		author := e.Selection.Find(".author, .byline, [rel='author']").First().Text()
		date := e.Selection.Find("time, .date, .published").First().Text()

		if title != "" {
			e.Item.Set("headline", title)
			if len(body) > 300 {
				body = body[:300] + "..."
			}
			e.Item.Set("preview", body)
			e.Item.Set("author", author)
			e.Item.Set("date", date)
		}
	})

	// Start crawling multiple news sites (robots.txt compliant)
	seeds := []string{
		"https://news.ycombinator.com",
		"https://www.techmeme.com",
		"https://dev.to",
	}

	if err := crawler.Start(seeds...); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	crawler.Wait()

	stats := crawler.Stats()
	fmt.Printf("\n✅ Complete! Articles: %v | Pages crawled: %v | Data: %v bytes\n",
		stats["items_scraped"], stats["requests_sent"], stats["bytes_downloaded"])
}
