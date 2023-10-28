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
	threadURL *url.URL
	title     string
	author    string
	startDate time.Time
	latest    time.Time
	replies   uint
	views     uint
}

func (rt RedditThread) URL() *url.URL        { return rt.threadURL }
func (rt RedditThread) Title() string        { return rt.title }
func (rt RedditThread) Author() string       { return rt.author }
func (rt RedditThread) StartDate() time.Time { return rt.startDate }
func (rt RedditThread) Latest() time.Time    { return rt.latest }
func (rt RedditThread) Replies() uint        { return rt.replies }
func (rt RedditThread) Views() uint          { return rt.views }

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
		fmt.Println("Failed to fetch %s: ", subreddit, err)
		return
	}

	for _, post := range harvest.Posts {
		threadScraper := NewThreadScraper(siteId, forumId, post, bot, fs.db)
		threadScraper.LoadCommentsSince(cutoff)
	}

	fs.db.SetForumLastScraped(forumId, time.Now())
}

/*---------------------------------------------------------------------------*/

type RedditComment struct {
	author    string
	published time.Time
	content   string
}

func (rc RedditComment) Author() string       { return rc.author }
func (rc RedditComment) Published() time.Time { return rc.published }
func (rc RedditComment) Content() string      { return rc.content }

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
		threadURL: permalink,
		author:    ts.post.Author,
		title:     ts.post.Name,
		startDate: time.Unix(int64(ts.post.CreatedUTC), 0),
		replies:   uint(ts.post.NumComments),
	}
	threadId := ts.db.InsertOrUpdateThread(ts.siteId, ts.forumId, thread)
	fmt.Printf("ThreadScraper %d loading comments from %s\n", threadId, permalink)

	post, err := ts.bot.Thread(ts.post.Permalink)
	if err != nil {
		log.Fatal(err)
	}
	for _, comment := range post.Replies {
		rc := RedditComment{
			author:    comment.Author,
			published: time.Unix(int64(comment.CreatedUTC), 0),
			content:   comment.Body,
		}
		ts.Comments = append(ts.Comments, rc)
	}

	comments := make([]model.Comment, len(ts.Comments), len(ts.Comments))
	for i := range ts.Comments {
		comments[i] = ts.Comments[i]
	}
	ts.db.AddComments(ts.siteId, threadId, comments)
}
