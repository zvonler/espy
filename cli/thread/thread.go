package thread

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath string
)

func NewCommand() *cobra.Command {
	threadCommand := &cobra.Command{
		Use:   "thread",
		Short: "Commands for searching threads",
		Args:  cobra.ExactArgs(2),
		Example: "  # Finds threads with comments containing 'Cybertruck'\n" +
			"  " + os.Args[0] + " thread grep Cybertruck",
	}

	threadCommand.AddCommand(initContentCommand())
	threadCommand.AddCommand(initGrepCommand())
	threadCommand.AddCommand(initListCommand())
	threadCommand.AddCommand(initParticipantsCommand())
	threadCommand.AddCommand(initPresentCommand())

	return threadCommand
}
