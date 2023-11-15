package thread

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/configuration"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/model"
)

func initContentCommand() *cobra.Command {
	contentCommand := &cobra.Command{
		Use:   "content <thread_id | thread_URL>",
		Short: "Prints the content of a thread",
		Args:  cobra.ExactArgs(1),
		Run:   runContentCommand,
	}
	return contentCommand
}

func runContentCommand(cmd *cobra.Command, args []string) {
	var err error
	var sdb *database.ScraperDB
	var thread model.Thread
	var comments []model.Comment

	if sdb, err = configuration.OpenExistingDatabase(); err == nil {
		defer sdb.Close()
		if thread, err = sdb.FindThread(args[0]); err == nil {
			if comments, err = sdb.ThreadComments(thread.Id); err == nil {
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
