package reddit

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/turnage/graw/reddit"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/model"
)

/*---------------------------------------------------------------------------*/

type RedditThread struct {
	model.Thread
}

/*---------------------------------------------------------------------------*/

type RedditComment struct {
	model.Comment
}

/*---------------------------------------------------------------------------*/

type ForumScraper struct {
	forumURL *url.URL
	db       *database.ScraperDB
	Threads  []RedditThread
}

func NewForumScraper(url *url.URL, db *database.ScraperDB) *ForumScraper {
	fs := new(ForumScraper)
	fs.forumURL = url
	fs.db = db
	return fs
}

func (fs *ForumScraper) LoadThreadsWithActivitySince(cutoff time.Time) {
	siteId, forumId := fs.db.InsertOrUpdateForum(fs.forumURL)

	subreddit := fs.forumURL.Path
	if !strings.HasPrefix(subreddit, "/r/") {
		panic(subreddit)
	}

	bot, err := reddit.NewBotFromAgentFile("reddit.agent", 0)
	if err != nil {
		log.Fatal(err)
	}
	harvest, err := bot.Listing(subreddit, "")
	if err != nil {
		fmt.Printf("Failed to fetch %s: %v\n", subreddit, err)
		return
	}

	for _, post := range harvest.Posts {
		threadScraper := NewThreadScraper(siteId, forumId, post, bot, fs.db)
		threadScraper.LoadCommentsSince(cutoff)
	}

	fs.db.SetForumLastScraped(forumId, time.Now())
}

/*---------------------------------------------------------------------------*/

type ThreadScraper struct {
	siteId   database.SiteID
	forumId  database.ForumID
	post     *reddit.Post
	bot      reddit.Bot
	db       *database.ScraperDB
	Comments []RedditComment
}

func NewThreadScraper(siteId database.SiteID, forumId database.ForumID, post *reddit.Post, bot reddit.Bot, db *database.ScraperDB) *ThreadScraper {
	ts := new(ThreadScraper)
	ts.siteId = siteId
	ts.forumId = forumId
	ts.post = post
	ts.bot = bot
	ts.db = db
	return ts
}

func (ts *ThreadScraper) LoadCommentsSince(cutoff time.Time) {
	permalink, err := url.Parse(ts.post.Permalink)
	if err != nil {
		log.Fatal(err)
	}
	thread := RedditThread{
		Thread: model.Thread{
			URL:       permalink,
			Author:    ts.post.Author,
			Title:     ts.post.Name,
			StartDate: time.Unix(int64(ts.post.CreatedUTC), 0),
			Replies:   uint(ts.post.NumComments),
		},
	}
	threadId := ts.db.InsertOrUpdateThread(ts.siteId, ts.forumId, thread.Thread)
	fmt.Printf("ThreadScraper %d loading comments from %s\n", threadId, permalink)

	post, err := ts.bot.Thread(ts.post.Permalink)
	if err != nil {
		log.Fatal(err)
	}
	for _, comment := range post.Replies {
		rc := RedditComment{
			Comment: model.Comment{
				Author:    comment.Author,
				Published: time.Unix(int64(comment.CreatedUTC), 0),
				Content:   comment.Body,
			},
		}
		ts.Comments = append(ts.Comments, rc)
	}

	comments := make([]model.Comment, len(ts.Comments), len(ts.Comments))
	for i := range ts.Comments {
		comments[i] = ts.Comments[i].Comment
	}
	ts.db.AddComments(ts.siteId, threadId, comments)
}
