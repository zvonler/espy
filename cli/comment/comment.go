package comment

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	commentCommand := &cobra.Command{
		Use:   "comment",
		Short: "Commands for searching and presenting comments",
	}

	commentCommand.AddCommand(initGrepCommand())

	return commentCommand
}
