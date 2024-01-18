package thread

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/configuration"
	"github.com/zvonler/espy/database"
)

func initGrepCommand() *cobra.Command {
	grepCommand := &cobra.Command{
		Use:   "grep <regex>...",
		Short: "Locates threads matching one or more regular expression(s)",
		Args:  cobra.MinimumNArgs(1),
		Run:   runGrepCommand,
	}

	grepCommand.Flags().StringVar(&startTime, "start-time", "", "Ignore comments before start-time")
	grepCommand.Flags().StringVar(&endTime, "end-time", "", "Ignore comments after end-time")

	return grepCommand
}

func runGrepCommand(cmd *cobra.Command, args []string) {
	var err error
	var sdb *database.ScraperDB

	if sdb, err = configuration.OpenExistingDatabase(); err != nil {
		log.Fatal(err)
	}

	defer sdb.Close()

	stmt := `
		SELECT DISTINCT
			t.id, t.title, t.url
		FROM thread t, comment c
		WHERE
			t.id = c.thread_id`

	dateTimeLayout := "20060102T15:04"

	if startTime != "" {
		startTm, _ := time.Parse(dateTimeLayout, startTime)
		stmt += ` AND c.published >= ` + strconv.FormatInt(startTm.Unix(), 10)
	}
	if endTime != "" {
		endTm, _ := time.Parse(dateTimeLayout, endTime)
		stmt += ` AND c.published < ` + strconv.FormatInt(endTm.Unix(), 10)
	}

	for _ = range args {
		stmt += " AND c.content REGEXP ?"
	}

	anyArgs := make([]any, len(args))
	for i := range args {
		anyArgs[i] = args[i]
	}

	sdb.ForEachRowOrPanic(
		func(rows *sql.Rows) {
			var id uint
			var title string
			var URL string
			rows.Scan(&id, &title, &URL)
			fmt.Printf("Thread %d: %q (%s)\n", id, title, URL)
		},
		stmt, anyArgs...)
}
