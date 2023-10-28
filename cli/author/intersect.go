package author

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/database"
)

var (
	dbPath string
)

func initIntersectCommand() *cobra.Command {
	intersectCommand := &cobra.Command{
		Use:   "intersect [-d DB] <query1> <query2>",
		Short: "Returns usernames that match both provided queries",
		Args:  cobra.ExactArgs(2),
		Run:   runIntersectCommand,
	}

	intersectCommand.Flags().StringVar(&dbPath, "database", "espy.db", "Database filename")

	return intersectCommand
}

func runIntersectCommand(cmd *cobra.Command, args []string) {
	authorQuery := "SELECT username FROM author WHERE "
	sql := fmt.Sprintf("%s %s INTERSECT %s %s", authorQuery, args[0], authorQuery, args[1])

	sdb := database.OpenScraperDB(dbPath)
	defer sdb.Close()
	rows, err := sdb.DB.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var username string
		rows.Scan(&username)
		fmt.Println(username)
	}
}
