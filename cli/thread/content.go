package thread

import (
	"fmt"
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
	sdb := database.OpenScraperDB(dbPath)
	defer sdb.Close()

	var digitCheck = regexp.MustCompile(`^[0-9]+$`)
	if digitCheck.MatchString(args[0]) {
		// Query by thread_id
		threadId, err := strconv.Atoi(args[0])
		if err != nil {
			panic(err)
		}

		sql := `SELECT content FROM comment c WHERE c.thread_id = ?`

		if rows, err := sdb.DB.Query(sql, threadId); err == nil {
			defer rows.Close()
			for rows.Next() {
				var content string
				rows.Scan(&content)
				fmt.Println(content)
			}
		} else {
			panic(err)
		}
	} else {
		url, err := url.Parse(args[0])
		if err != nil {
			panic(err)
		}
		url = utils.TrimmedURL(url)

		// Query by thread URL
		sql := `
			SELECT
				content
			FROM comment c, thread t
			WHERE
				    c.thread_id = t.id
				AND t.url = ?`

		if rows, err := sdb.DB.Query(sql, url.String()); err == nil {
			defer rows.Close()
			for rows.Next() {
				var content string
				rows.Scan(&content)
				fmt.Println(content)
			}
		} else {
			panic(err)
		}
	}
}
