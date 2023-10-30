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
)

func initPresentCommand() *cobra.Command {
	presentCommand := &cobra.Command{
		Use:   "present [-d DB] [thread_id | URL]",
		Short: "Formats the content of a thread for human consumption",
		Args:  cobra.MinimumNArgs(1),
		Run:   runPresentCommand,
	}

	presentCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")

	return presentCommand
}

func runPresentCommand(cmd *cobra.Command, args []string) {
	sdb, err := database.OpenScraperDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sdb.Close()

	var threadId database.ThreadID

	var digitCheck = regexp.MustCompile(`^[0-9]+$`)
	if digitCheck.MatchString(args[0]) {
		// Query by thread_id
		if id, err := strconv.Atoi(args[0]); err == nil {
			threadId = database.ThreadID(id)
		} else {
			panic(err)
		}
	} else {
		// Look up thread id by URL
		url, err := url.Parse(args[0])
		if err != nil {
			log.Fatalf("Failed to parse thread URL: %v", err)
		}
		_, threadId, err = sdb.GetThreadByURL(url)
		if err != nil {
			log.Fatal(err)
		}
	}

	printRows := func(rows *sql.Rows) {
		var url string
		var username string
		var content string
		rows.Scan(&url, &username, &content)
		fmt.Printf("%s\n%s: %q\n", url, username, content)
		fmt.Println("--------")
	}

	stmt := `SELECT c.url, a.username, c.content FROM comment c, author a WHERE a.id = c.author_id AND c.thread_id = ? ORDER BY published`
	sdb.ForEachRowOrPanic(printRows, stmt, threadId)
}
