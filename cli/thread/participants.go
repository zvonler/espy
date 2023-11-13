package thread

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
)

func initParticipantsCommand() *cobra.Command {
	participantsCommand := &cobra.Command{
		Use:   "participants <thread_id | thread_URL>",
		Short: "Lists the authors that have commented in a thread",
		Args:  cobra.ExactArgs(1),
		Run:   runParticipantsCommand,
	}

	participantsCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")

	return participantsCommand
}

func runParticipantsCommand(cmd *cobra.Command, args []string) {
	var err error

	if sdb, err := database.OpenScraperDB(dbPath); err == nil {
		defer sdb.Close()
		if thread, err := sdb.FindThread(args[0]); err == nil {
			if usernames, err := sdb.ThreadParticipants(thread.Id); err == nil {
				for _, username := range usernames {
					fmt.Println(username)
				}
			}
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}
