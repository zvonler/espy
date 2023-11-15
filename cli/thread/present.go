package thread

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/bit101/go-ansi"
	"github.com/spf13/cobra"
	"github.com/zvonler/espy/configuration"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/model"
	"golang.org/x/term"
)

func initPresentCommand() *cobra.Command {
	presentCommand := &cobra.Command{
		Use:   "present <thread_id | thread_URL>",
		Short: "Formats the content of a thread for human consumption",
		Args:  cobra.MinimumNArgs(1),
		Run:   runPresentCommand,
	}
	return presentCommand
}

func paginateComments(thread model.Thread, comments []model.Comment) {
	cmd := exec.Command("/usr/bin/less", "-FRX")
	cmd.Stdout = os.Stdout

	if stdin, err := cmd.StdinPipe(); err == nil {
		go func() {
			defer stdin.Close()

			ansi.Fprintf(stdin, ansi.Yellow, "%s", thread.Title)
			ansi.Fprintf(stdin, ansi.Default, " by ")
			ansi.Fprintf(stdin, ansi.Red, "%s", thread.Author)
			ansi.Fprintf(stdin, ansi.Default, " (")
			ansi.Fprintf(stdin, ansi.Cyan, "%s", thread.URL)
			ansi.Fprintf(stdin, ansi.Default, ")\n")
			ansi.Fprintln(stdin, ansi.Blue, "========")

			for _, c := range comments {
				ansi.Fprintf(stdin, ansi.Cyan, "%s\n", c.URL)
				ansi.Fprintf(stdin, ansi.Red, "%s", c.Author)
				ansi.Fprintf(stdin, ansi.Default, ": ")
				ansi.Fprintf(stdin, ansi.Green, "\"")
				ansi.Fprintf(stdin, ansi.Default, c.Content)
				ansi.Fprintf(stdin, ansi.Green, "\"\n")
				ansi.Fprintln(stdin, ansi.Blue, "--------")
			}
		}()
	} else {
		log.Fatal(err)
	}

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func printComments(thread model.Thread, comments []model.Comment) {
	fmt.Printf("%s: (%s)\n", thread.Title, thread.URL)
	for _, c := range comments {
		fmt.Printf("%s\n%s: %q\n", c.URL, c.Author, c.Content)
		fmt.Println("--------")
	}
}

func runPresentCommand(cmd *cobra.Command, args []string) {
	isTty := term.IsTerminal(int(os.Stdout.Fd()))

	var err error
	var sdb *database.ScraperDB
	var thread model.Thread
	var comments []model.Comment

	if sdb, err = configuration.OpenExistingDatabase(); err == nil {
		defer sdb.Close()
		if thread, err = sdb.FindThread(args[0]); err == nil {
			if comments, err = sdb.ThreadComments(thread.Id); err == nil {
				if isTty {
					paginateComments(thread, comments)
				} else {
					printComments(thread, comments)
				}
			}
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}
