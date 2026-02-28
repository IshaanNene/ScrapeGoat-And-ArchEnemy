// GitHub Trending Scraper — scrapes today's trending repos
// Extracts: repo name, description, language, stars, forks, stars gained today
package main

import (
	"fmt"
	"strings"
	"time"

	webstalk "github.com/IshaanNene/ScrapeGoat-And-ArchEnemy/pkg/webstalk"
)

func main() {
	fmt.Println("⭐ GitHub Trending Scraper")
	fmt.Println("   Extracting trending repos: name, description, language, stars, forks")

	crawler := webstalk.NewCrawler(
		webstalk.WithConcurrency(3),
		webstalk.WithMaxDepth(0), // Single page
		webstalk.WithDelay(1*time.Second),
		webstalk.WithOutput("json", "./output/github"),
		webstalk.WithAllowedDomains("github.com"),
		webstalk.WithMaxRequests(5),
	)

	// Extract each trending repo
	crawler.OnHTML("article.Box-row", func(e *webstalk.Element) {
		repoPath := strings.TrimSpace(e.Selection.Find("h2 a").Text())
		repoPath = strings.Join(strings.Fields(repoPath), "") // Remove whitespace
		description := strings.TrimSpace(e.Selection.Find("p.col-9").Text())
		language := strings.TrimSpace(e.Selection.Find("[itemprop='programmingLanguage']").Text())

		// Stars and forks
		links := e.Selection.Find("div.f6 a")
		stars := ""
		forks := ""
		if links.Length() > 0 {
			stars = strings.TrimSpace(links.First().Text())
		}
		if links.Length() > 1 {
			forks = strings.TrimSpace(links.Eq(1).Text())
		}

		// Stars today
		starsToday := strings.TrimSpace(e.Selection.Find("span.d-inline-block.float-sm-right").Text())

		if repoPath != "" {
			e.Item.Set("repo", repoPath)
			e.Item.Set("description", description)
			e.Item.Set("language", language)
			e.Item.Set("stars", stars)
			e.Item.Set("forks", forks)
			e.Item.Set("stars_today", starsToday)
			e.Item.Set("url", "https://github.com/"+repoPath)
		}
	})

	if err := crawler.Start(
		"https://github.com/trending",
		"https://github.com/trending?since=weekly",
		"https://github.com/trending/go",
		"https://github.com/trending/python",
		"https://github.com/trending/rust",
	); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	crawler.Wait()

	stats := crawler.Stats()
	fmt.Printf("\n✅ Complete! Repos scraped: %v | Pages: %v\n",
		stats["items_scraped"], stats["requests_sent"])
}
