package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/cli/scrape"
)

func NewCommand() *cobra.Command {
	espyCli := &cobra.Command{
		Use:     "espy",
		Short:   "Espy CLI",
		Long:    "Espy Command Line Interface",
		Example: fmt.Sprintf("  %s <command> [flags...]", os.Args[0]),
	}

	espyCli.AddCommand(scrape.NewCommand())

	return espyCli
}
