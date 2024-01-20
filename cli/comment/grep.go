package comment

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bit101/go-ansi"
	"github.com/spf13/cobra"
	"github.com/zvonler/espy/configuration"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/model"
	"golang.org/x/term"
)

func initGrepCommand() *cobra.Command {
	grepCommand := &cobra.Command{
		Use:   "grep [-d DB] <regex>...",
		Short: "Locates comments matching one or more regular expression(s)",
		Args:  cobra.MinimumNArgs(1),
		Run:   runGrepCommand,
	}

	// !@# compile error, why?
	//grepCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")

	return grepCommand
}

func paginateComments(comments []model.Comment) {
	cmd := exec.Command("/usr/bin/less", "-FRX")
	cmd.Stdout = os.Stdout

	if stdin, err := cmd.StdinPipe(); err == nil {
		go func() {
			defer stdin.Close()

			for _, c := range comments {
				ansi.Fprintf(stdin, ansi.Cyan, "%s ", c.URL)
				ansi.Fprintf(stdin, ansi.Green, "%s\n", c.Published)
				ansi.Fprintf(stdin, ansi.Red, "%s", c.Author)
				ansi.Fprintf(stdin, ansi.Default, ": ")
				ansi.Fprintf(stdin, ansi.Green, "\"")
				ansi.Fprintf(stdin, ansi.Default, "%s", c.Content)
				ansi.Fprintf(stdin, ansi.Green, "\"\n")
				ansi.Fprintln(stdin, ansi.Blue, "--------")
			}
		}()
	} else {
		log.Fatal(err)
	}

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func printComments(comments []model.Comment) {
	for _, c := range comments {
		fmt.Printf("%s %s\n%s: %q\n", c.URL, c.Published, c.Author, c.Content)
		fmt.Println("--------")
	}

}

func runGrepCommand(cmd *cobra.Command, args []string) {
	var err error
	var sdb *database.ScraperDB
	var comments []model.Comment

	if sdb, err = configuration.OpenExistingDatabase(); err == nil {
		defer sdb.Close()

		stmt := `
		SELECT
			c.url, a.username, c.published, c.content
		FROM author a, comment c
		WHERE
				a.id = c.author_id`

		exprs := make([]string, len(args))
		anyArgs := make([]any, len(args))
		for i := range args {
			e := "AND c.content REGEXP ?"
			exprs = append(exprs, e)
			anyArgs[i] = args[i]
		}
		stmt = stmt + " " + strings.Join(exprs, " ")

		stmt += `
			ORDER BY published DESC`

		sdb.ForEachRowOrPanic(
			func(rows *sql.Rows) {
				var urlStr string
				var username string
				var published uint
				var content string
				err = rows.Scan(&urlStr, &username, &published, &content)
				if err != nil {
					panic(err)
				}
				url, _ := url.Parse(urlStr)
				comments = append(comments,
					model.Comment{
						URL:       url,
						Author:    username,
						Published: time.Unix(int64(published), 0),
						Content:   content,
					})
			}, stmt, anyArgs...)

		isTty := term.IsTerminal(int(os.Stdout.Fd()))
		if isTty {
			paginateComments(comments)
		} else {
			printComments(comments)
		}
	}
}
