package thread

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
)

func initGrepCommand() *cobra.Command {
	grepCommand := &cobra.Command{
		Use:   "grep [-d DB] <regex>...",
		Short: "Locates threads matching one or more regular expression(s)",
		Args:  cobra.MinimumNArgs(1),
		Run:   runGrepCommand,
	}

	grepCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")

	return grepCommand
}

func runGrepCommand(cmd *cobra.Command, args []string) {
	sdb, err := database.OpenScraperDB(dbPath)
	if err != nil {
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
		func(rows *sql.Rows) bool {
			var id uint
			var title string
			var URL string
			rows.Scan(&id, &title, &URL)
			fmt.Printf("Thread %d: %q (%s)\n", id, title, URL)
			return true
		},
		stmt, anyArgs...)
}
