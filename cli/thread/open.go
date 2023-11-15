package thread

import (
	"log"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/zvonler/espy/configuration"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/model"
)

func initOpenCommand() *cobra.Command {
	openCommand := &cobra.Command{
		Use:   "open <thread_id | thread_URL>",
		Short: "Opens a thread in a browser.",
		Args:  cobra.ExactArgs(1),
		Run:   runOpenCommand,
	}
	return openCommand
}

func runOpenCommand(cmd *cobra.Command, args []string) {
	var err error
	var sdb *database.ScraperDB
	var thread model.Thread

	if sdb, err = configuration.OpenExistingDatabase(); err == nil {
		defer sdb.Close()
		if thread, err = sdb.FindThread(args[0]); err == nil {
			browser.OpenURL(thread.URL.String())
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}
