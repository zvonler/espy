package xf_scraper

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

type ForumScraper struct {
	forumURL  *url.URL
	db        *ScraperDB
	Threads   []Thread
	SubForums []*url.URL
	collector *colly.Collector
}

func NewForumScraper(forumURL *url.URL, db *ScraperDB) *ForumScraper {
	fs := new(ForumScraper)
	fs.Threads = make([]Thread, 0)
	fs.collector = newCollectorWithCFRoundtripper()
	fs.db = db

	if strings.HasSuffix(forumURL.RequestURI(), "/") {
		// Eliminate trailing slashes to canonicalize URL for database
		if trimmed, err := url.Parse(strings.TrimRight(forumURL.String(), "/")); err == nil {
			fs.forumURL = trimmed
		} else {
			log.Fatalf("Bad URL: %v", err)
		}
	} else {
		fs.forumURL = forumURL
	}

	fs.collector.OnHTML("div.node--forum", func(e *colly.HTMLElement) {
		e.ForEach("h3.node-title", func(_ int, e *colly.HTMLElement) {
			if url, err := url.Parse(e.ChildAttr("a", "href")); err == nil {
				fs.SubForums = append(fs.SubForums, e.Request.URL.ResolveReference(url))
			} else {
				log.Printf("Failed to parse href for %v\n", e)
			}
		})
	})

	fs.collector.OnHTML("div.mark-thread:not([class*=is-prefix])", func(e *colly.HTMLElement) {
		temp := Thread{}
		temp.Author = e.Attr("data-author")

		e.ForEach("div.structItem-title", func(_ int, e *colly.HTMLElement) {
			temp.Title = e.ChildText("a")
			if threadHref, err := url.Parse(e.ChildAttr("a", "href")); err == nil {
				temp.URL = e.Request.URL.ResolveReference(threadHref)
			}
		})

		e.ForEach("li.structItem-startDate", func(_ int, e *colly.HTMLElement) {
			dataTime := e.ChildAttr("time.u-dt", "data-time")
			if tm, err := strconv.Atoi(dataTime); err != nil {
				log.Printf("Unparseable data-time '%v' for %s", dataTime, temp.Title)
			} else {
				temp.StartDate = time.Unix(int64(tm), 0)
			}
		})

		e.ForEach("div.structItem-cell--meta", func(_ int, e *colly.HTMLElement) {
			e.ForEach("dl.pairs", func(_ int, e *colly.HTMLElement) {
				dt := e.ChildText("dt")
				dd := e.ChildText("dd")
				if dt == "Replies" {
					temp.Replies = parseCompactCount(dd)
				} else if dt == "Views" {
					temp.Views = parseCompactCount(dd)
				}
			})
		})

		e.ForEach("div.structItem-cell--latest", func(_ int, e *colly.HTMLElement) {
			dataTime := e.ChildAttr("time.u-dt", "data-time")
			if tm, err := strconv.Atoi(dataTime); err != nil {
				log.Printf("Unparseable data-time '%v' for %s", dataTime, temp.Title)
			} else {
				temp.Latest = time.Unix(int64(tm), 0)
			}
		})

		fs.Threads = append(fs.Threads, temp)
	})

	fs.collector.OnRequest(func(r *colly.Request) {
		fmt.Println("ForumScraper visiting", r.URL.String())
	})

	fs.collector.OnError(func(r *colly.Response, err error) {
		fmt.Printf("ForumScraper got %v for %s\n", err, r.Request.URL)
	})

	return fs
}

func (fs *ForumScraper) LoadThreadsWithActivitySince(cutoff time.Time) {
	siteId, forumId := fs.db.InsertOrUpdateForum(fs.forumURL)

	fs.collector.Visit(fs.forumURL.String())
	time.Sleep(1 + time.Duration(rand.Intn(3))*time.Second)

	if len(fs.Threads) > 0 {
		for pageNum := 2; fs.Threads[len(fs.Threads)-1].Latest.After(cutoff); pageNum++ {
			time.Sleep(1 + time.Duration(rand.Intn(4))*time.Second)
			next := fs.forumURL.JoinPath(fmt.Sprintf("page-%d", pageNum))
			fs.collector.Visit(next.String())
		}

		for _, thread := range fs.Threads {
			ts := NewThreadScraper(siteId, forumId, thread, fs.db)
			ts.LoadCommentsSince(cutoff)
		}
	}

	if len(fs.SubForums) > 0 {
		for _, subForumURL := range fs.SubForums {
			sfs := NewForumScraper(subForumURL, fs.db)
			sfs.LoadThreadsWithActivitySince(cutoff)
		}
	}

	fs.db.SetForumLastScraped(forumId, time.Now())
}

func parseCompactCount(c string) (res uint) {
	if len(c) > 0 {
		switch c[len(c)-1] {
		case 'K':
			c = c[:len(c)-1] + "000"
		case 'M':
			c = c[:len(c)-1] + "000000"
		}
		if val, err := strconv.ParseUint(c, 10, 32); err == nil {
			res = uint(val)
		}
	}
	return
}