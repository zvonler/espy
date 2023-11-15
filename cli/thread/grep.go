package thread

import (
	"database/sql"
	"fmt"
	"log"

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
