package parser

import (
	"github.com/IshaanNene/ScrapeGoat-And-ArchEnemy/internal/config"
	"github.com/IshaanNene/ScrapeGoat-And-ArchEnemy/internal/types"
)

// Parser extracts data and links from a fetched response.
type Parser interface {
	// Parse extracts items and follow-up URLs from a response.
	// It returns scraped items, discovered links, and any error.
	Parse(resp *types.Response, rules []config.ParseRule) ([]*types.Item, []string, error)
}
