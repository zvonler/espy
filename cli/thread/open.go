package thread

import (
	"log"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
)

func initOpenCommand() *cobra.Command {
	openCommand := &cobra.Command{
		Use:   "open <thread_id | thread_URL>",
		Short: "Opens a thread in a browser.",
		Args:  cobra.ExactArgs(1),
		Run:   runOpenCommand,
	}
	openCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")
	return openCommand
}

func runOpenCommand(cmd *cobra.Command, args []string) {
	var err error

	if sdb, err := database.OpenScraperDB(dbPath); err == nil {
		defer sdb.Close()
		if thread, err := sdb.FindThread(args[0]); err == nil {
			browser.OpenURL(thread.URL.String())
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}
