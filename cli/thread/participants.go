package thread

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/configuration"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/model"
)

func initParticipantsCommand() *cobra.Command {
	participantsCommand := &cobra.Command{
		Use:   "participants <thread_id | thread_URL>",
		Short: "Lists the authors that have commented in a thread",
		Args:  cobra.ExactArgs(1),
		Run:   runParticipantsCommand,
	}
	return participantsCommand
}

func runParticipantsCommand(cmd *cobra.Command, args []string) {
	var err error
	var sdb *database.ScraperDB
	var thread model.Thread
	var usernames []string

	if sdb, err = configuration.OpenExistingDatabase(); err == nil {
		defer sdb.Close()
		if thread, err = sdb.FindThread(args[0]); err == nil {
			if usernames, err = sdb.ThreadParticipants(thread.Id); err == nil {
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
