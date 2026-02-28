// Wikipedia Knowledge Extractor — deep crawl with 1000 article limit
// Extracts: title, summary, categories, reference count, external links
package main

import (
	"fmt"
	"strings"
	"time"

	scrapegoat "github.com/IshaanNene/ScrapeGoat/pkg/scrapegoat"
)

func main() {
	fmt.Println("📚 Wikipedia Knowledge Extractor")
	fmt.Println("   Deep crawl: title, summary, categories, references")

	crawler := scrapegoat.NewCrawler(
		scrapegoat.WithConcurrency(10),
		scrapegoat.WithMaxDepth(3),
		scrapegoat.WithDelay(100*time.Millisecond),
		scrapegoat.WithOutput("jsonl", "./output/wikipedia"),
		scrapegoat.WithAllowedDomains("en.wikipedia.org"),
		scrapegoat.WithMaxRequests(1000),
		scrapegoat.WithRobotsRespect(true),
	)

	// Follow internal wiki links
	crawler.OnHTML("#bodyContent a[href^='/wiki/']", func(e *scrapegoat.Element) {
		href := e.Attr("href")
		if !strings.Contains(href, ":") {
			e.Follow("https://en.wikipedia.org" + href)
		}
	})

	// Extract article data
	crawler.OnHTML("#content", func(e *scrapegoat.Element) {
		title := strings.TrimSpace(e.Selection.Find("#firstHeading").Text())
		if title == "" {
			return
		}

		// First paragraph
		firstP := strings.TrimSpace(e.Selection.Find("#mw-content-text .mw-parser-output > p").First().Text())
		if len(firstP) > 500 {
			firstP = firstP[:500] + "..."
		}

		// Category text
		catText := strings.TrimSpace(e.Selection.Find("#mw-normal-catlinks ul").Text())
		var categories []string
		if catText != "" {
			for _, c := range strings.Split(catText, "\n") {
				c = strings.TrimSpace(c)
				if c != "" {
					categories = append(categories, c)
				}
			}
		}

		refCount := e.Selection.Find(".reflist li").Length()
		extCount := e.Selection.Find(".external.text").Length()
		imgCount := e.Selection.Find("#mw-content-text img").Length()

		e.Item.Set("title", title)
		e.Item.Set("summary", firstP)
		if len(categories) > 0 {
			e.Item.Set("categories", categories)
		}
		e.Item.Set("references", refCount)
		e.Item.Set("external_links", extCount)
		e.Item.Set("images", imgCount)
	})

	if err := crawler.Start(
		"https://en.wikipedia.org/wiki/Web_scraping",
		"https://en.wikipedia.org/wiki/Artificial_intelligence",
		"https://en.wikipedia.org/wiki/Go_(programming_language)",
		"https://en.wikipedia.org/wiki/Distributed_computing",
		"https://en.wikipedia.org/wiki/Machine_learning",
	); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	crawler.Wait()

	stats := crawler.Stats()
	fmt.Printf("\n✅ Complete! Articles: %v | Pages crawled: %v | Data: %v bytes\n",
		stats["items_scraped"], stats["requests_sent"], stats["bytes_downloaded"])
}
