package thread

import (
	"fmt"
	"log"
	"math"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/model"
)

func initListCommand() *cobra.Command {
	listCommand := &cobra.Command{
		Use:   "list",
		Short: "Lists threads in the database",
		Run:   runListCommand,
	}

	listCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")
	return listCommand
}

func runListCommand(cmd *cobra.Command, args []string) {
	sdb, err := database.OpenScraperDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sdb.Close()

	if threadsById, err := sdb.GetThreads([]model.ThreadID{}); err == nil {
		colWidth := uint(math.Round(math.Ceil(math.Log10(float64(len(threadsById))))))
		fmtString := fmt.Sprintf("%%0%dd: %%s (%%s)\n", colWidth)
		for id, thread := range threadsById {
			fmt.Printf(fmtString, id, thread.Title, thread.URL)
		}
	}

	if err != nil {
		log.Fatal(err)
	}

}
