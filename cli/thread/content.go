package thread

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
)

func initContentCommand() *cobra.Command {
	contentCommand := &cobra.Command{
		Use:   "content [-d DB] <thread_id | thread_URL>",
		Short: "Prints the content of a thread",
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
		if threadId, err := sdb.FindThread(args[0]); err == nil {
			if comments, err := sdb.ThreadComments(threadId); err == nil {
				for _, comment := range comments {
					fmt.Println(comment.Content)
				}
			}
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}
