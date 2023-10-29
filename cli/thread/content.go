package thread

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/utils"
)

func initContentCommand() *cobra.Command {
	contentCommand := &cobra.Command{
		Use:   "content [-d DB] [thread_id | URL]",
		Short: "Prints the content of a thread",
		Args:  cobra.MinimumNArgs(1),
		Run:   runContentCommand,
	}

	contentCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")

	return contentCommand
}

func runContentCommand(cmd *cobra.Command, args []string) {
	sdb, err := database.OpenScraperDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sdb.Close()

	printRows := func(rows *sql.Rows) {
		var content string
		rows.Scan(&content)
		fmt.Println(content)
	}

	var digitCheck = regexp.MustCompile(`^[0-9]+$`)
	if digitCheck.MatchString(args[0]) {
		// Query by thread_id
		threadId, err := strconv.Atoi(args[0])
		if err != nil {
			panic(err)
		}

		stmt := `SELECT content FROM comment c WHERE c.thread_id = ? ORDER BY published`
		sdb.ForEachRowOrPanic(printRows, stmt, threadId)
	} else {
		url, err := url.Parse(args[0])
		if err != nil {
			log.Fatalf("Failed to parse thread URL: %v", err)
		}
		url = utils.TrimmedURL(url)

		// Query by thread URL
		stmt := `
			SELECT
				content
			FROM comment c, thread t
			WHERE
				    c.thread_id = t.id
				AND t.url = ?
			ORDER BY published`

		sdb.ForEachRowOrPanic(printRows, stmt, url.String())
	}
}
