package scrape

import (
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/reddit"
	"github.com/zvonler/espy/xf_scraper"
)

var (
	lookbackDays int
	dbPath       string
)

func NewCommand() *cobra.Command {
	scrapeCommand := &cobra.Command{
		Use:   "scrape [-d DB] <URL>",
		Short: "Scrape forums and threads",
		Args:  cobra.ExactArgs(1),
		Example: "" +
			"  " + os.Args[0] + " scrape https://site.com/forum-url",
		Run: runScrapeCommand,
	}

	scrapeCommand.Flags().IntVar(&lookbackDays, "lookback-days", 7, "Ignore activity earlier than lookback-days before now")
	scrapeCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")

	return scrapeCommand
}

func runScrapeCommand(cmd *cobra.Command, args []string) {
	url, err := url.Parse(args[0])
	if err != nil {
		log.Fatalf("Bad URL: %v", err)
	}

	db := database.OpenScraperDB(dbPath)
	defer db.Close()

	cutoff := time.Now().AddDate(0, 0, -lookbackDays)

	if strings.Contains(url.Host, "reddit.com") {
		fs := reddit.NewForumScraper(url, db)
		fs.LoadThreadsWithActivitySince(cutoff)
	} else if strings.Contains(url.Path, "/forums/") {
		fs := xf_scraper.NewForumScraper(url, db)
		fs.LoadThreadsWithActivitySince(cutoff)
	}
}
