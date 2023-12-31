package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/vartanbeno/go-reddit/v2/reddit"
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
	siteId, forumId, err := fs.db.InsertOrUpdateForum(fs.forumURL)
	if err != nil {
		panic(err)
	}

	subreddit := fs.forumURL.Path
	if !strings.HasPrefix(subreddit, "/r/") {
		panic(subreddit)
	}
	subreddit = strings.TrimPrefix(subreddit, "/r/")

	content, err := ioutil.ReadFile("reddit.agent")
	if err != nil {
		log.Fatal(err)
	}
	var credentials reddit.Credentials
	err = json.Unmarshal(content, &credentials)
	if err != nil {
		log.Fatal("Error unmarshaling JSON")
	}

	client, err := reddit.NewClient(credentials)
	if err != nil {
		fmt.Printf("Failed to fetch %s: %v\n", subreddit, err)
		return
	}

	posts, _, err := client.Subreddit.NewPosts(context.Background(), subreddit, &reddit.ListOptions{
		Limit: 300,
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, post := range posts {
		postAndComments, _, err := client.Post.Get(context.Background(), post.ID)
		if err != nil {
			log.Fatal(err)
		}
		threadScraper := NewThreadScraper(siteId, forumId, postAndComments, fs.db)
		threadScraper.LoadCommentsSince(cutoff)
	}

	fs.db.SetForumLastScraped(forumId, time.Now())
}

/*---------------------------------------------------------------------------*/

type ThreadScraper struct {
	siteId   model.SiteID
	forumId  model.ForumID
	post     *reddit.PostAndComments
	db       *database.ScraperDB
	Comments []RedditComment
}

func NewThreadScraper(siteId model.SiteID, forumId model.ForumID, post *reddit.PostAndComments, db *database.ScraperDB) *ThreadScraper {
	ts := new(ThreadScraper)
	ts.siteId = siteId
	ts.forumId = forumId
	ts.post = post
	ts.db = db
	return ts
}

func (ts *ThreadScraper) LoadCommentsSince(cutoff time.Time) {
	permalink, err := url.Parse("https://reddit.com" + ts.post.Post.Permalink)
	if err != nil {
		log.Fatal(err)
	}
	thread := RedditThread{
		Thread: model.Thread{
			URL:       permalink,
			Author:    ts.post.Post.Author,
			Title:     ts.post.Post.Title,
			StartDate: ts.post.Post.Created.Time,
			Replies:   uint(ts.post.Post.NumberOfComments),
		},
	}
	threadId, err := ts.db.InsertOrUpdateThread(ts.siteId, ts.forumId, thread.Thread)
	if err != nil {
		panic(err)
	}
	fmt.Printf("ThreadScraper %d loading comments from %s\n", threadId, permalink)

	var toRc func(c *reddit.Comment)

	toRc = func(c *reddit.Comment) {
		permalink, err := url.Parse("https://reddit.com" + c.Permalink)
		if err != nil {
			log.Fatal(err)
		}
		ts.Comments = append(ts.Comments, RedditComment{
			Comment: model.Comment{
				URL:       permalink,
				Author:    c.Author,
				Published: c.Created.Time,
				Content:   c.Body,
			},
		})

		for _, r := range c.Replies.Comments {
			toRc(r)
		}
	}

	for _, comment := range ts.post.Comments {
		toRc(comment)
	}

	comments := make([]model.Comment, len(ts.Comments), len(ts.Comments))
	for i := range ts.Comments {
		comments[i] = ts.Comments[i].Comment
	}
	ts.db.AddComments(ts.siteId, threadId, comments)
}
