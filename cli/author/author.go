package author

import (
	"os"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	authorCommand := &cobra.Command{
		Use:   "author",
		Short: "Commands for searching authors",
		Args:  cobra.ExactArgs(2),
		Example: "  # Finds authors with comments containing 'Cybertruck'\n" +
			"  " + os.Args[0] + " author grep Cybertruck",
	}

	authorCommand.AddCommand(initContentCommand())
	authorCommand.AddCommand(initGrepCommand())
	authorCommand.AddCommand(initIntersectCommand())

	return authorCommand
}
