package xf_scraper

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/gocolly/colly"
)

type ThreadPageFinder struct {
	thread          XFThread
	pageNumFinder   *colly.Collector
	earliestScraped time.Time
	latestScraped   time.Time
}

func NewThreadPageFinder(t XFThread) *ThreadPageFinder {
	tpf := new(ThreadPageFinder)
	tpf.thread = t
	tpf.pageNumFinder = newCollectorWithCFRoundtripper()

	tpf.pageNumFinder.OnHTML("article.message--post", func(e *colly.HTMLElement) {
		e.ForEach("ul.message-attribution-main", func(_ int, e *colly.HTMLElement) {
			dataTime := e.ChildAttr("time.u-dt", "data-time")
			if tm, err := strconv.Atoi(dataTime); err != nil {
				log.Printf("Unparseable data-time '%v'", dataTime)
			} else {
				published := time.Unix(int64(tm), 0)
				if published.After(tpf.latestScraped) {
					tpf.latestScraped = published
				}
				if tpf.earliestScraped.IsZero() || published.Before(tpf.earliestScraped) {
					tpf.earliestScraped = published
				}
			}
		})
	})

	tpf.pageNumFinder.OnRequest(func(r *colly.Request) {
		fmt.Printf("ThreadPageFinder visiting %s\n", r.URL.String())
	})

	tpf.pageNumFinder.OnError(func(r *colly.Response, err error) {
		fmt.Printf("ThreadPageFinder got %v with body %s\n", err, r.Body)
	})

	return tpf
}

// Returns the highest page number containing comments earlier than time, or zero
func (tpf *ThreadPageFinder) FindCommentsBefore(target time.Time, pages uint) uint {
	if pages == 1 {
		tpf.earliestScraped, tpf.latestScraped = time.Time{}, time.Time{}
		tpf.pageNumFinder.Visit(tpf.thread.pageURL(1).String())
		if tpf.earliestScraped.Before(target) {
			return 1
		}
		return 0
	}

	// Binary search between the endpoints
	left, right := uint(1), pages
	for left < right {
		mid := left + (right-left)/2
		tpf.earliestScraped, tpf.latestScraped = time.Time{}, time.Time{}
		tpf.pageNumFinder.Visit(tpf.thread.pageURL(mid).String())
		if tpf.earliestScraped.Before(target) && tpf.latestScraped.After(target) {
			return mid
		}
		if tpf.earliestScraped.Before(target) {
			left = mid + 1
		} else {
			right = mid - 1
		}
		time.Sleep(1 + time.Duration(rand.Intn(6))*time.Second)
	}
	return left
}
