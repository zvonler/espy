package thread

import (
	"fmt"
	"log"

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
	var err error

	if sdb, err := database.OpenScraperDB(dbPath); err == nil {
		defer sdb.Close()
		if threadId, err := sdb.FindThread(args[0]); err == nil {
			if comments, err := sdb.ThreadComments(threadId); err == nil {
				for _, c := range comments {
					fmt.Printf("%s\n%s: %q\n", c.URL, c.Author, c.Content)
					fmt.Println("--------")
				}
			}
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}
