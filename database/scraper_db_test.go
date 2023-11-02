package database

import (
	"database/sql"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zvonler/espy/model"
)

func TestBasicDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	db, err := OpenScraperDB(tmpDir + "/test.db")
	require.Equal(t, nil, err)
	defer db.Close()

	firstLoaded := db.FirstCommentLoaded(ThreadID(0))
	require.Equal(t, false, firstLoaded)

	forumHref := "https://some-forum.com/forums/name.123"
	forumUrl, err := url.Parse(forumHref)
	require.Equal(t, nil, err)
	siteId, forumId, err := db.InsertOrUpdateForum(forumUrl)
	require.Equal(t, nil, err)
	require.Greater(t, siteId, SiteID(0))
	require.Greater(t, forumId, ForumID(0))

	{
		// Test that trailing '/' is considered equal
		forumUrl, err := url.Parse(forumHref + "/")
		require.Equal(t, nil, err)
		altSiteId, altForumId, err := db.InsertOrUpdateForum(forumUrl)
		require.Equal(t, nil, err)
		require.Equal(t, siteId, altSiteId)
		require.Equal(t, forumId, altForumId)
	}

	threadUrl, err := url.Parse("https://some-forum.com/forums/name.123/thread-xyz")
	require.Equal(t, nil, err)

	thread := model.Thread{
		Title: "Some thread",
		URL:   threadUrl,
	}
	threadId, err := db.InsertOrUpdateThread(siteId, forumId, thread)
	require.Equal(t, nil, err)
	require.Greater(t, threadId, ThreadID(0))

	{
		dbSiteId, dbThreadId, err := db.GetThreadByURL(threadUrl)
		require.Equal(t, nil, err)
		require.Equal(t, siteId, dbSiteId)
		require.Equal(t, dbThreadId, threadId)
	}

	require.Equal(t, []time.Time(nil), db.CommentTimeRange(threadId))

	commentUrl, err := url.Parse("https://some-forum.com/forums/name.123/thread-xyz/comments/foo")
	require.Equal(t, nil, err)

	author := "somebody"
	published := time.Unix(123456789, 0)
	commentBody := "Some text"
	comment := model.Comment{
		Author:    author,
		Published: published,
		Content:   commentBody,
		URL:       commentUrl,
	}
	err = db.AddComments(siteId, threadId, []model.Comment{comment})
	require.Equal(t, nil, err)

	times := db.CommentTimeRange(threadId)
	require.Equal(t, 2, len(times))
	require.Equal(t, published, times[0])
	require.Equal(t, published, times[1])

	scrapeTime := time.Now().Truncate(time.Second)
	db.SetForumLastScraped(forumId, scrapeTime)
	tm, err := db.GetForumLastScraped(forumId)
	require.Equal(t, err, nil)
	require.Equal(t, scrapeTime, tm)

	findAuthor := func(pattern string) (found bool) {
		db.ForSingleRowOrPanic(
			func(rows *sql.Rows) {
				var dbAuthor string
				var dbPublished int64
				var dbBody string
				rows.Scan(&dbAuthor, &dbPublished, &dbBody)
				require.Equal(t, author, dbAuthor)
				require.Equal(t, published, time.Unix(dbPublished, 0))
				require.Equal(t, commentBody, dbBody)
				found = true
			},
			"SELECT a.username, c.published, c.content FROM comment c, author a WHERE "+
				"a.username REGEXP ? "+
				"AND a.id = c.author_id",
			pattern)
		return
	}

	// Confirm regex support
	require.True(t, findAuthor("^somebody$"))
	require.False(t, findAuthor("^SOMEBODY$"))
	require.True(t, findAuthor("(?i)^SOMEBODY$"))
	require.True(t, findAuthor("[[:alpha:]]{8}"))
	require.False(t, findAuthor("[[:digit:]]"))
}
