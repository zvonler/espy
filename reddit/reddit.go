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
	Threads  []RedditThread
	client   *reddit.Client
}

func NewForumScraper(url *url.URL) *ForumScraper {
	fs := new(ForumScraper)
	fs.forumURL = url

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
		fmt.Printf("Failed to create client: %v\n", err)
		return nil
	}
	fs.client = client

	return fs
}

func (fs *ForumScraper) SubredditPostsSince(cutoff time.Time) (posts []*reddit.Post, err error) {
	subreddit := fs.forumURL.Path
	if !strings.HasPrefix(subreddit, "/r/") {
		panic(subreddit)
	}
	subreddit = strings.TrimPrefix(subreddit, "/r/")

	posts, _, err = fs.client.Subreddit.NewPosts(context.Background(), subreddit, &reddit.ListOptions{
		Limit: 300,
	})
	return
}

func (fs *ForumScraper) LoadThreadsWithActivitySince(db *database.ScraperDB, cutoff time.Time) {
	posts, err := fs.SubredditPostsSince(cutoff)
	if err != nil {
		log.Fatal(err)
	}

	siteId, forumId, err := db.InsertOrUpdateForum(fs.forumURL)
	if err != nil {
		panic(err)
	}

	for _, post := range posts {
		postAndComments, _, err := fs.client.Post.Get(context.Background(), post.ID)
		if err != nil {
			log.Fatal(err)
		}
		threadScraper := NewThreadScraper(siteId, forumId, postAndComments)
		threadScraper.LoadCommentsSince(db, cutoff)
	}

	db.SetForumLastScraped(forumId, time.Now())
}

/*---------------------------------------------------------------------------*/

type ThreadScraper struct {
	siteId   model.SiteID
	forumId  model.ForumID
	post     *reddit.PostAndComments
	Comments []RedditComment
}

func NewThreadScraper(siteId model.SiteID, forumId model.ForumID, post *reddit.PostAndComments) *ThreadScraper {
	ts := new(ThreadScraper)
	ts.siteId = siteId
	ts.forumId = forumId
	ts.post = post
	return ts
}

func (ts *ThreadScraper) LoadCommentsSince(db *database.ScraperDB, cutoff time.Time) {
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
	threadId, err := db.InsertOrUpdateThread(ts.siteId, ts.forumId, thread.Thread)
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
	db.AddComments(ts.siteId, threadId, comments)
}
