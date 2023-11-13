package thread

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
)

var (
	untag bool
)

func initTagCommand() *cobra.Command {
	tagCommand := &cobra.Command{
		Use:   "tag [-u] <thread_id | thread_URL> TAGNAME...",
		Short: "Adds or removes a tag or tags from a thread.",
		Args:  cobra.MinimumNArgs(2),
		Run:   runTagCommand,
	}

	tagCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")
	tagCommand.Flags().BoolVar(&untag, "untag", false, "Remove tags")

	return tagCommand
}

func runTagCommand(cmd *cobra.Command, args []string) {
	var err error

	if sdb, err := database.OpenScraperDB(dbPath); err == nil {
		defer sdb.Close()
		if thread, err := sdb.FindThread(args[0]); err == nil {
			if untag {
				err = sdb.RemoveThreadTags(thread.Id, args[1:])
			} else {
				err = sdb.AddThreadTags(thread.Id, args[1:])
			}
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}
