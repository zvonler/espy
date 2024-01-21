package parse

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bit101/go-ansi"
	"github.com/spf13/cobra"
	"github.com/zvonler/espy/model"
	"github.com/zvonler/espy/reddit"
	"github.com/zvonler/espy/xf_scraper"
	"golang.org/x/term"
)

func NewCommand() *cobra.Command {
	parseCommand := &cobra.Command{
		Use:   "parse <URL>",
		Short: "Parse a single URL and describe its contents",
		Args:  cobra.ExactArgs(1),
		Example: "" +
			"  " + os.Args[0] + " parse https://site.com/forum-url",
		Run: runParseCommand,
	}

	return parseCommand
}

func paginateThreads(threads []xf_scraper.XFThread) {
	cmd := exec.Command("/usr/bin/less", "-FRX")
	cmd.Stdout = os.Stdout

	if stdin, err := cmd.StdinPipe(); err == nil {
		go func() {
			defer stdin.Close()

			for _, t := range threads {
				ansi.Fprintf(stdin, ansi.Green, "%s\n", t.Title)
				ansi.Fprintf(stdin, ansi.Red, "%s ", t.Author)
				ansi.Fprintf(stdin, ansi.Green, "%s ", t.StartDate)
				ansi.Fprintf(stdin, ansi.Yellow, "%s ", t.Latest)
				ansi.Fprintf(stdin, ansi.Purple, "(%d)\n", t.Replies)
				ansi.Fprintf(stdin, ansi.Cyan, "%s\n", t.URL)
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

func printThreads(threads []xf_scraper.XFThread) {
	for _, t := range threads {
		fmt.Printf("%s\n", t.URL)
		fmt.Println("--------")
	}
}

func paginateComments(comments []xf_scraper.XFComment) {
	cmd := exec.Command("/usr/bin/less", "-FRX")
	cmd.Stdout = os.Stdout

	if stdin, err := cmd.StdinPipe(); err == nil {
		go func() {
			defer stdin.Close()

			for _, c := range comments {
				ansi.Fprintf(stdin, ansi.Red, "%s ", c.Author)
				ansi.Fprintf(stdin, ansi.Green, "%s\n", c.Published)
				ansi.Fprintf(stdin, ansi.Default, "%s\n", c.Content)
				ansi.Fprintf(stdin, ansi.Cyan, "%s\n", c.URL)
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

func printComments(comments []xf_scraper.XFComment) {
	for _, c := range comments {
		fmt.Printf("%s %s\n%s: %q\n", c.URL, c.Published, c.Author, c.Content)
		fmt.Println("--------")
	}
}

func runParseCommand(cmd *cobra.Command, args []string) {
	url, err := url.Parse(args[0])
	if err != nil {
		log.Fatalf("Bad URL: %v", err)
	}

	isTty := term.IsTerminal(int(os.Stdout.Fd()))

	if strings.Contains(url.Host, "reddit.com") {
		fs := reddit.NewForumScraper(url)
		cutoff := time.Now().AddDate(0, 0, -7)
		posts, err := fs.SubredditPostsSince(cutoff)
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range posts {
			fmt.Println(p.Title)
		}
	} else if strings.Contains(url.Path, "/forums/") {
		fs := xf_scraper.NewForumScraper(url)
		fs.Collector.Visit(url.String())
		for _, thread := range fs.Threads {
			fmt.Println(thread)
		}
		if isTty {
			paginateThreads(fs.Threads)
		} else {
			printThreads(fs.Threads)
		}
	} else if strings.Contains(url.Path, "/threads/") {
		xfThread := xf_scraper.XFThread{model.Thread{URL: url}}
		ts := xf_scraper.NewThreadScraper(0, xfThread)
		ts.CommentScraper.Visit(url.String())

		if isTty {
			paginateComments(ts.Comments)
		} else {
			printComments(ts.Comments)
		}
	}
}
