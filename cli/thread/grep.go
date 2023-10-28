package thread

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/xf_scraper"
)

var (
	database string
)

func initGrepCommand() *cobra.Command {
	grepCommand := &cobra.Command{
		Use:   "grep [-d DB] <regex>...",
		Short: "Locates threads matching one or more regular expression(s)",
		Args:  cobra.MinimumNArgs(1),
		Run:   runGrepCommand,
	}

	grepCommand.Flags().StringVar(&database, "database", "espy.db", "Database filename")

	return grepCommand
}

func runGrepCommand(cmd *cobra.Command, args []string) {
	sdb := xf_scraper.OpenScraperDB(database)
	defer sdb.Close()

	filters := []string{}
	for _, arg := range args {
		filters = append(filters, fmt.Sprintf("c.content REGEXP %q", arg))
	}
	filterStr := strings.Join(filters, " AND ")

	sql := `
		SELECT DISTINCT
			t.id, t.title, t.url
		FROM thread t, comment c
		WHERE
			    t.id = c.thread_id
		AND `
	sql = sql + filterStr

	rows, err := sdb.DB.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uint
		var title string
		var URL string
		rows.Scan(&id, &title, &URL)
		fmt.Printf("Thread %d: %q (%s)\n", id, title, URL)
	}
}
