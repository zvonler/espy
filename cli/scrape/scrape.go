package scrape

import (
	"log"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/xf_scraper"
)

var (
	lookbackDays int
)

func NewCommand() *cobra.Command {
	scrapeCommand := &cobra.Command{
		Use:   "scrape",
		Short: "Scrape forums and threads",
		Args:  cobra.ExactArgs(2),
		Example: "" +
			"  " + os.Args[0] + " scrape https://site.com/forum-url espy.db",
		Run: runScrapeCommand,
	}

	scrapeCommand.Flags().IntVar(&lookbackDays, "lookback-days", 7, "Ignore activity earlier than lookback-days before now")

	return scrapeCommand
}

func runScrapeCommand(cmd *cobra.Command, args []string) {
	url, err := url.Parse(args[0])
	if err != nil {
		log.Fatalf("Bad URL: %v", err)
	}

	db := database.OpenScraperDB(args[1])
	defer db.Close()

	fs := xf_scraper.NewForumScraper(url, db)
	cutoff := time.Now().AddDate(0, 0, -lookbackDays)
	fs.LoadThreadsWithActivitySince(cutoff)
}
