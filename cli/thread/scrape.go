package thread

import (
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/model"
	"github.com/zvonler/espy/xf_scraper"
)

var (
	lookbackDays int
)

func initScrapeCommand() *cobra.Command {
	scrapeCommand := &cobra.Command{
		Use:   "scrape [-d DB] [thread_id | URL]",
		Short: "Scrapes a threads comments",
		Args:  cobra.MinimumNArgs(1),
		Run:   runScrapeCommand,
	}

	scrapeCommand.Flags().IntVar(&lookbackDays, "lookback-days", 720, "Ignore activity earlier than lookback-days before now")
	scrapeCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")

	return scrapeCommand
}

func runScrapeCommand(cmd *cobra.Command, args []string) {
	cutoff := time.Now().AddDate(0, 0, -lookbackDays)
	if sdb, err := database.OpenScraperDB(dbPath); err == nil {
		defer sdb.Close()
		if thread, err := sdb.FindThread(args[0]); err == nil {
			xfThread := xf_scraper.XFThread{model.Thread{URL: thread.URL}}
			ts := xf_scraper.NewThreadScraper(thread.Id, xfThread, sdb)
			ts.LoadCommentsSince(cutoff)
			comments := make([]model.Comment, len(ts.Comments), len(ts.Comments))
			for i := range ts.Comments {
				comments[i] = ts.Comments[i].Comment
			}
			sdb.AddComments(thread.SiteId, thread.Id, comments)
		}
	} else {
		log.Fatal(err)
	}
}
