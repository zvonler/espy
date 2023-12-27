package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zvonler/espy/cli/author"
	"github.com/zvonler/espy/cli/comment"
	"github.com/zvonler/espy/cli/forum"
	"github.com/zvonler/espy/cli/scrape"
	"github.com/zvonler/espy/cli/site"
	"github.com/zvonler/espy/cli/thread"
)

var (
	dbPath string
)

func NewCommand() *cobra.Command {
	espyCli := &cobra.Command{
		Use:     "espy",
		Short:   "Espy CLI",
		Long:    "Espy Command Line Interface",
		Example: fmt.Sprintf("  %s <command> [flags...]", os.Args[0]),
	}

	espyCli.PersistentFlags().StringVar(&dbPath, "database", "espy.db", "Database filename")
	viper.BindPFlag("database", espyCli.PersistentFlags().Lookup("database"))

	espyCli.AddCommand(author.NewCommand())
	espyCli.AddCommand(comment.NewCommand())
	espyCli.AddCommand(forum.NewCommand())
	espyCli.AddCommand(scrape.NewCommand())
	espyCli.AddCommand(site.NewCommand())
	espyCli.AddCommand(thread.NewCommand())

	return espyCli
}
