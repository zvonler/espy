package forum

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath string
)

func NewCommand() *cobra.Command {
	forumCommand := &cobra.Command{
		Use:   "forum",
		Short: "Commands for searching forums",
		Example: "  # List forums\n" +
			"  " + os.Args[0] + " forum list",
	}

	forumCommand.AddCommand(initListCommand())

	return forumCommand
}
