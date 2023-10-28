package thread

import (
	"os"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	threadCommand := &cobra.Command{
		Use:   "thread",
		Short: "Commands for searching threads",
		Args:  cobra.ExactArgs(2),
		Example: "  # Finds threads with comments containing 'Cybertruck'\n" +
			"  " + os.Args[0] + " thread grep Cybertruck",
	}

	threadCommand.AddCommand(initGrepCommand())

	return threadCommand
}
