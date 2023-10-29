package forum

import (
	"database/sql"
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

	stmt := `SELECT id, url FROM forum ORDER BY URL`

	type Forum struct {
		id  uint
		url string
	}
	forums := make([]Forum, 0)

	sdb.ForEachRowOrPanic(
		func(rows *sql.Rows) {
			var id uint
			var url string
			rows.Scan(&id, &url)
			forums = append(forums, Forum{id, url})
		},
		stmt)

	colWidth := uint(math.Round(math.Ceil(math.Log10(float64(len(forums))))))
	fmtString := fmt.Sprintf("%%0%dd: %%s\n", colWidth)
	for _, forum := range forums {
		fmt.Printf(fmtString, forum.id, forum.url)
	}
}
