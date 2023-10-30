package site

import (
	"github.com/spf13/cobra"
)

var (
	dbPath string
)

func NewCommand() *cobra.Command {
	siteCommand := &cobra.Command{
		Use:   "site",
		Short: "Commands for working with sites",
	}

	siteCommand.AddCommand(initUpdateCommand())

	return siteCommand
}
