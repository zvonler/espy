package site

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
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
)

func initUpdateCommand() *cobra.Command {
	updateCommand := &cobra.Command{
		Use:   "update",
		Short: "Updates all known forums at a site",
		Run:   runUpdateCommand,
	}

	updateCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")
	updateCommand.Flags().IntVar(&lookbackDays, "lookback-days", 7, "Ignore activity earlier than lookback-days before now")

	return updateCommand
}

func runUpdateCommand(cmd *cobra.Command, args []string) {
	sdb, err := database.OpenScraperDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sdb.Close()

	var siteId model.SiteID

	var digitCheck = regexp.MustCompile(`^[0-9]+$`)
	if digitCheck.MatchString(args[0]) {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			panic(err)
		} else {
			siteId = model.SiteID(id)
		}
	} else {
		siteId, err = sdb.GetSiteId(args[0])
		if err != nil {
			log.Fatal(err)
		}
	}

	cutoff := time.Now().AddDate(0, 0, -lookbackDays)

	var urls []*url.URL

	stmt := "SELECT url FROM forum WHERE site_id = ?"
	sdb.ForEachRowOrPanic(
		func(rows *sql.Rows) {
			var urlStr string
			rows.Scan(&urlStr)
			if url, err := url.Parse(urlStr); err != nil {
				log.Fatal(err)
			} else {
				urls = append(urls, url)
			}
		},
		stmt, siteId)

	for i, url := range urls {
		fmt.Printf("%d: %s\n", i, url)
		if strings.Contains(url.Host, "reddit.com") {
			fs := reddit.NewForumScraper(url)
			fs.LoadThreadsWithActivitySince(sdb, cutoff)
		} else if strings.Contains(url.Path, "/forums/") {
			fs := xf_scraper.NewForumScraper(url)
			fs.LoadThreadsWithActivitySince(sdb, cutoff, false)
		}
	}
}
