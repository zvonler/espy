package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zvonler/espy/cli/author"
	"github.com/zvonler/espy/cli/forum"
	"github.com/zvonler/espy/cli/scrape"
	"github.com/zvonler/espy/cli/thread"
)

func NewCommand() *cobra.Command {
	espyCli := &cobra.Command{
		Use:     "espy",
		Short:   "Espy CLI",
		Long:    "Espy Command Line Interface",
		Example: fmt.Sprintf("  %s <command> [flags...]", os.Args[0]),
	}

	espyCli.AddCommand(author.NewCommand())
	espyCli.AddCommand(forum.NewCommand())
	espyCli.AddCommand(scrape.NewCommand())
	espyCli.AddCommand(thread.NewCommand())

	return espyCli
}
