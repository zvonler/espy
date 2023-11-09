package author

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ryanuber/columnize"
	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
)

func initGrepCommand() *cobra.Command {
	grepCommand := &cobra.Command{
		Use:   "grep [-d DB] <regex>...",
		Short: "Locates author usernames matching one or more regular expression(s)",
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
		SELECT
			a.id, a.username, COUNT(c.id) comments, MAX(c.published) latest
		FROM author a, comment c
		WHERE
			c.author_id = a.id`

	exprs := make([]string, len(args))
	anyArgs := make([]any, len(args))
	for i := range args {
		e := "AND a.username REGEXP ?"
		exprs = append(exprs, e)
		anyArgs[i] = args[i]
	}
	stmt = stmt + " " + strings.Join(exprs, " ")

	stmt += `
		GROUP BY a.id, a.username
		ORDER BY latest DESC, comments`

	output := []string{
		"AuthorID | Username | Comments | Latest",
	}

	sdb.ForEachRowOrPanic(
		func(rows *sql.Rows) {
			var id uint
			var username string
			var comments uint
			var latestTm int64
			rows.Scan(&id, &username, &comments, &latestTm)
			latest := time.Unix(latestTm, 0)
			output = append(output, fmt.Sprintf("%d | %s | %d | %v", id, username, comments, latest))
		},
		stmt, anyArgs...)

	fmt.Println(columnize.SimpleFormat(output))
}
