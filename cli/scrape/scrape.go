package scrape

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/model"
	"github.com/zvonler/espy/reddit"
	"github.com/zvonler/espy/xf_scraper"
)

var (
	lookbackDays int
	dbPath       string
	noChanges    bool
)

func NewCommand() *cobra.Command {
	scrapeCommand := &cobra.Command{
		Use:   "scrape <URL>",
		Short: "Scrape forums and threads",
		Args:  cobra.ExactArgs(1),
		Example: "" +
			"  " + os.Args[0] + " scrape https://site.com/forum-url",
		Run: runScrapeCommand,
	}

	scrapeCommand.Flags().IntVar(&lookbackDays, "lookback-days", 7, "Ignore activity earlier than lookback-days before now")
	scrapeCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")
	scrapeCommand.Flags().BoolVar(&noChanges, "no-changes", false, "Make no changes to the database")

	return scrapeCommand
}

func runScrapeCommand(cmd *cobra.Command, args []string) {
	url, err := url.Parse(args[0])
	if err != nil {
		log.Fatalf("Bad URL: %v", err)
	}

	sdb, err := database.OpenScraperDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sdb.Close()

	cutoff := time.Now().AddDate(0, 0, -lookbackDays)

	if strings.Contains(url.Host, "reddit.com") {
		fs := reddit.NewForumScraper(url)
		fs.LoadThreadsWithActivitySince(sdb, cutoff)
	} else if strings.Contains(url.Path, "/forums/") {
		fs := xf_scraper.NewForumScraper(url)
		fs.LoadThreadsWithActivitySince(sdb, cutoff, true)
	} else if strings.Contains(url.Path, "/threads/") {
		// If url already in thread table, create ThreadScraper
		if thread, err := sdb.GetThreadByURL(url); err == nil {
			xfThread := xf_scraper.XFThread{model.Thread{URL: thread.URL}}
			ts := xf_scraper.NewThreadScraper(thread.Id, xfThread)
			ts.LoadCommentsSince(sdb, cutoff)
			comments := make([]model.Comment, len(ts.Comments), len(ts.Comments))
			for i := range ts.Comments {
				comments[i] = ts.Comments[i].Comment
			}
			if !noChanges {
				sdb.AddComments(thread.SiteId, thread.Id, comments)
			} else {
				for _, c := range comments {
					fmt.Println(c.URL.String())
				}
			}
		} else {
			// Else get forum from thread page?
			panic("Can't load new thread without forum and site\n")
		}
	}
}
