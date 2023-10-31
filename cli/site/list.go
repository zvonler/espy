package site

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
		Short: "Lists sites in the database",
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

	if sitesById, err := sdb.GetSites(); err == nil {
		colWidth := uint(math.Round(math.Ceil(math.Log10(float64(len(sitesById))))))
		fmtString := fmt.Sprintf("%%0%dd: %%s\n", colWidth)
		for id, hostname := range sitesById {
			fmt.Printf(fmtString, id, hostname)
		}
	}

	if err != nil {
		log.Fatal(err)
	}

}
