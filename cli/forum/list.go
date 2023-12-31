package forum

import (
	"fmt"
	"log"
	"math"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
)

func initListCommand() *cobra.Command {
	listCommand := &cobra.Command{
		Use:   "list",
		Short: "Lists forums in the database",
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

	if forumsByUrl, err := sdb.GetForums(); err == nil {
		colWidth := uint(math.Round(math.Ceil(math.Log10(float64(len(forumsByUrl))))))
		fmtString := fmt.Sprintf("%%0%dd: %%s\n", colWidth)
		for id, url := range forumsByUrl {
			fmt.Printf(fmtString, id, url)
		}
	}

	if err != nil {
		log.Fatal(err)
	}

}
