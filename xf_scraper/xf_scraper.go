package xf_scraper

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/caffix/cloudflare-roundtripper/cfrt"
	"github.com/gocolly/colly"
)

type Thread struct {
	URL       *url.URL
	Title     string
	Author    string
	StartDate time.Time
	Replies   uint
	Latest    time.Time
	Views     uint
}

func (t Thread) pageURL(pageNum uint) *url.URL {
	if pageNum == 1 {
		return t.URL
	}
	return t.URL.JoinPath(fmt.Sprintf("page-%d", pageNum))
}

type Comment struct {
	Author    string
	Published time.Time
	Content   string
}

func newCollectorWithCFRoundtripper() *colly.Collector {
	collector := colly.NewCollector()

	collector = colly.NewCollector(
		colly.IgnoreRobotsTxt(),
		colly.UserAgent("Mozilla"),
	)
	transport, err :=
		cfrt.New(&http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   15 * time.Second,
				KeepAlive: 15 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		})
	if err != nil {
		log.Fatal(err)
	}
	collector.WithTransport(transport)
	collector.Limit(&colly.LimitRule{
		Parallelism: 1,
		RandomDelay: 10 * time.Second,
	})
	return collector
}
