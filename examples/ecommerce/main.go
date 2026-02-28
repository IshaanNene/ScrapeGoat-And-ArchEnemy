// E-Commerce Product Scraper — scrapes books.toscrape.com
// Extracts: title, price, rating, availability, description, image
package main

import (
	"fmt"
	"strings"
	"time"

	scrapegoat "github.com/IshaanNene/ScrapeGoat/pkg/scrapegoat"
)

func main() {
	fmt.Println("🛒 E-Commerce Product Scraper — books.toscrape.com")
	fmt.Println("   Extracting products: title, price, rating, stock, image")

	crawler := scrapegoat.NewCrawler(
		scrapegoat.WithConcurrency(5),
		scrapegoat.WithMaxDepth(3),
		scrapegoat.WithDelay(200*time.Millisecond),
		scrapegoat.WithOutput("json", "./output/ecommerce"),
		scrapegoat.WithAllowedDomains("books.toscrape.com"),
		scrapegoat.WithMaxRequests(200),
	)

	// Follow pagination and category links
	crawler.OnHTML("li.next a[href]", func(e *scrapegoat.Element) {
		e.Follow(e.Attr("href"))
	})
	crawler.OnHTML(".side_categories ul li a", func(e *scrapegoat.Element) {
		e.Follow(e.Attr("href"))
	})

	// Follow individual product links
	crawler.OnHTML("h3 a[href]", func(e *scrapegoat.Element) {
		e.Follow(e.Attr("href"))
	})

	// Extract product details from individual product pages
	crawler.OnHTML(".product_page", func(e *scrapegoat.Element) {
		title := e.Selection.Find("h1").Text()
		price := e.Selection.Find(".price_color").First().Text()
		availability := strings.TrimSpace(e.Selection.Find(".availability").First().Text())
		description := strings.TrimSpace(e.Selection.Find("#product_description ~ p").Text())
		img, _ := e.Selection.Find("#product_gallery img").Attr("src")

		// Star rating from class
		starClass, _ := e.Selection.Find(".star-rating").Attr("class")
		rating := "Unknown"
		for _, r := range []string{"One", "Two", "Three", "Four", "Five"} {
			if strings.Contains(starClass, r) {
				rating = r
				break
			}
		}

		// Breadcrumb category
		category := e.Selection.Find(".breadcrumb li").Eq(2).Find("a").Text()

		if title != "" {
			e.Item.Set("title", title)
			e.Item.Set("price", price)
			e.Item.Set("rating", rating)
			e.Item.Set("availability", availability)
			e.Item.Set("image", img)
			e.Item.Set("category", strings.TrimSpace(category))
			if len(description) > 200 {
				description = description[:200] + "..."
			}
			e.Item.Set("description", description)
		}
	})

	if err := crawler.Start("https://books.toscrape.com"); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	crawler.Wait()

	stats := crawler.Stats()
	fmt.Printf("\n✅ Complete! Requests: %v | Products scraped: %v | Data: %v bytes\n",
		stats["requests_sent"], stats["items_scraped"], stats["bytes_downloaded"])
}
