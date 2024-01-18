package author

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
)

func initContentCommand() *cobra.Command {
	contentCommand := &cobra.Command{
		Use:   "content [-d DB] <username>",
		Short: "Prints the content of an author's comments",
		Args:  cobra.ExactArgs(1),
		Run:   runContentCommand,
	}

	contentCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")

	return contentCommand
}

func runContentCommand(cmd *cobra.Command, args []string) {
	var err error

	if sdb, err := database.OpenScraperDB(dbPath); err == nil {
		defer sdb.Close()
		if comments, err := sdb.FindAuthorComments(args[0]); err == nil {
			for _, comment := range comments {
				fmt.Println(comment.URL)
				fmt.Println(comment.Content)
			}
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}
