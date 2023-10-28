package xf_scraper

import (
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/zvonler/espy/database"
	"golang.org/x/net/html"
)

type ThreadScraper struct {
	threadId        database.ThreadID
	thread          XFThread
	db              *database.ScraperDB
	Comments        []XFComment
	pages           uint
	commentScraper  *colly.Collector
	pageNumScraper  *colly.Collector
	earliestScraped time.Time
	latestScraped   time.Time
}

func NewThreadScraper(threadId database.ThreadID, thread XFThread, db *database.ScraperDB) *ThreadScraper {
	ts := new(ThreadScraper)
	ts.threadId = threadId
	ts.thread = thread
	ts.db = db
	ts.Comments = make([]XFComment, 0)
	ts.pages = 1

	ts.pageNumScraper = newCollectorWithCFRoundtripper()
	ts.pageNumScraper.OnHTML("nav.pageNavWrapper--mixed", func(e *colly.HTMLElement) {
		// Pages with nav bars have one at top and at bottom, so skip the second
		if ts.pages > 1 {
			return
		}

		e.ForEach("ul.pageNav-main", func(_ int, e *colly.HTMLElement) {
			var lastPage string
			e.ForEach("a", func(_ int, e *colly.HTMLElement) {
				lastPage = e.Text
			})
			if intPages, err := strconv.Atoi(lastPage); err == nil {
				ts.pages = uint(intPages)
			} else {
				fmt.Printf("Couldn't parse lastPage from %v\n", lastPage)
			}
		})
	})

	ts.pageNumScraper.OnRequest(func(r *colly.Request) {
		fmt.Printf("PageNumScraper (%d) visiting %s\n", ts.threadId, r.URL.String())
	})

	ts.pageNumScraper.OnError(func(r *colly.Response, err error) {
		fmt.Printf("PageNumScraper got %v with body %s\n", err, r.Body)
	})

	ts.commentScraper = newCollectorWithCFRoundtripper()
	ts.commentScraper.OnHTML("article.message--post", func(e *colly.HTMLElement) {
		temp := XFComment{}
		temp.Author = e.Attr("data-author")
		e.ForEach("article.message-body", func(_ int, e *colly.HTMLElement) {
			// These get just the content of the blockquote
			// temp.Content = e.DOM.ChildrenFiltered(".bbCodeBlock--quote").Text()
			// temp.Content = e.ChildText(".bbCodeBlock--quote")

			// These approaches return the full text including blockquotes
			// temp.Content = e.DOM.ChildrenFiltered("*").Text()
			// temp.Content = e.DOM.ChildrenFiltered("*:not(.bbCodeBlock--quote)").Text()
			// temp.Content = e.DOM.Not("div").Text()
			// temp.Content = e.DOM.Not(".bbCodeBlock--quote").Text()
			// temp.Content = e.Text

			// These double up the text including the blockquote contents
			// temp.Content = e.ChildText("*:not(.bbCodeBlock--quote)")
			// temp.Content = e.ChildText("*")

			// This returns no text
			// temp.Content = e.DOM.Not("*").Text()

			// Approaches above don't work so filter blockquotes manually
			outer, _ := goquery.OuterHtml(e.DOM)
			doc, _ := html.Parse(strings.NewReader(outer))
			var collectText func(*html.Node)
			collectText = func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == "blockquote" {
					return
				}
				if n.Type == html.TextNode {
					temp.Content += n.Data
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					collectText(c)
				}
			}
			collectText(doc)
		})

		// Replace non-breaking spaces with regular spaces
		temp.Content = strings.ReplaceAll(temp.Content, "\u00a0", " ")

		// Remove whitespace-only lines
		wsLinePat := regexp.MustCompile("\n[ \t]+\n")
		temp.Content = string(wsLinePat.ReplaceAll([]byte(temp.Content), []byte("\n")))

		// Replace repeated newlines with singles
		nlPat := regexp.MustCompile("\n\n+")
		temp.Content = string(nlPat.ReplaceAll([]byte(temp.Content), []byte("\n")))

		// Trim leading and trailing newlines
		temp.Content = strings.TrimLeft(temp.Content, "\n")
		temp.Content = strings.TrimRight(temp.Content, "\n")

		e.ForEach("ul.message-attribution-main", func(_ int, e *colly.HTMLElement) {
			dataTime := e.ChildAttr("time.u-dt", "data-time")
			if tm, err := strconv.Atoi(dataTime); err != nil {
				log.Printf("Unparseable data-time '%v' for %s", dataTime, temp.Author)
			} else {
				temp.Published = time.Unix(int64(tm), 0)
				if temp.Published.After(ts.latestScraped) {
					ts.latestScraped = temp.Published
				}
				if ts.earliestScraped.IsZero() || temp.Published.Before(ts.earliestScraped) {
					ts.earliestScraped = temp.Published
				}
			}
		})

		ts.Comments = append(ts.Comments, temp)
	})

	ts.commentScraper.OnRequest(func(r *colly.Request) {
		fmt.Printf("CommentScraper (%d) visiting %s\n", ts.threadId, r.URL.String())
	})

	ts.commentScraper.OnError(func(r *colly.Response, err error) {
		fmt.Printf("CommentScraper (%d) got %v with body %s\n", ts.threadId, err, r.Body)
	})

	return ts
}

func (ts *ThreadScraper) LoadCommentsSince(cutoff time.Time) {
	if timeRange := ts.db.CommentTimeRange(ts.threadId); timeRange != nil {
		// If the database already has some comments for this thread, avoid
		// re-loading them.
		earliest, latest := timeRange[0], timeRange[1]

		if ts.thread.Latest != latest {
			// Loading the first page of the thread gets us the last page number
			ts.pageNumScraper.Visit(ts.thread.URL.String())
			time.Sleep(1 + time.Duration(rand.Intn(2))*time.Second)

			// Load from last page until earlier than the latest already loaded
			for pageNum := ts.pages; pageNum >= 1; pageNum-- {
				time.Sleep(1 + time.Duration(rand.Intn(4))*time.Second)
				next := ts.thread.pageURL(pageNum)
				ts.commentScraper.Visit(next.String())
				if ts.earliestScraped.Before(latest) {
					break
				}
			}
		}

		// Binary search to page containing posts older than earliest then load if before cutoff
		if cutoff.Before(earliest) {

			// If we already have the first comment of the thread, don't look for more
			if !ts.db.FirstCommentLoaded(ts.threadId) {

				// Loading the first page of the thread gets us the last page number
				ts.pageNumScraper.Visit(ts.thread.URL.String())
				time.Sleep(1 + time.Duration(rand.Intn(2))*time.Second)
				tpf := NewThreadPageFinder(ts.thread)
				for pageNum := tpf.FindCommentsBefore(earliest, ts.pages); pageNum >= 1; pageNum-- {
					time.Sleep(1 + time.Duration(rand.Intn(4))*time.Second)
					next := ts.thread.pageURL(pageNum)
					ts.commentScraper.Visit(next.String())
					if ts.earliestScraped.Before(cutoff) {
						break
					}
				}
			}
		}
	} else {
		// Loading the first page of the thread gets us the last page number
		ts.pageNumScraper.Visit(ts.thread.URL.String())

		// Load from last page until earlier than cutoff or out of comments
		for pageNum := ts.pages; pageNum >= 1; pageNum-- {
			time.Sleep(1 + time.Duration(rand.Intn(4))*time.Second)
			next := ts.thread.pageURL(pageNum)
			ts.commentScraper.Visit(next.String())
			if ts.earliestScraped.Before(cutoff) {
				break
			}
		}
	}
}
