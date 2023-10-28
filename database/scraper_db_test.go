package database

import (
	"net/url"
	"os"
	"path/filepath"
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
	siteId, forumId := db.InsertOrUpdateForum(forumUrl)
	require.Greater(t, siteId, SiteID(0))
	require.Greater(t, forumId, ForumID(0))

	{
		// Test that trailing '/' is considered equal
		forumUrl, err := url.Parse(forumHref + "/")
		require.Equal(t, nil, err)
		altSiteId, altForumId := db.InsertOrUpdateForum(forumUrl)
		require.Equal(t, siteId, altSiteId)
		require.Equal(t, forumId, altForumId)
	}

	threadUrl, err := url.Parse("https://some-forum.com/forums/name.123/thread-xyz")
	require.Equal(t, nil, err)

	thread := model.Thread{
		Title: "Some thread",
		URL:   threadUrl,
	}
	threadId := db.InsertOrUpdateThread(siteId, forumId, thread)
	require.Greater(t, threadId, ThreadID(0))

	require.Equal(t, []time.Time(nil), db.CommentTimeRange(threadId))

	author := "somebody"
	published := time.Unix(123456789, 0)
	commentBody := "Some text"
	comment := model.Comment{
		Author:    author,
		Published: published,
		Content:   commentBody,
	}
	db.AddComments(siteId, threadId, []model.Comment{comment})

	times := db.CommentTimeRange(threadId)
	require.Equal(t, 2, len(times))
	require.Equal(t, published, times[0])
	require.Equal(t, published, times[1])
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	stat, err := exists(tmpDir)
	require.Equal(t, nil, err)
	require.Equal(t, true, stat)

	stat, err = exists(tmpDir + "/non-existent-path")
	require.Equal(t, nil, err)
	require.Equal(t, false, stat)

	subdir := filepath.Join(tmpDir, "unreadable")
	err = os.MkdirAll(subdir, 0700)
	require.Equal(t, nil, err)

	hiddenFile := filepath.Join(subdir, "somefile.tgz")
	fd, err := os.Create(hiddenFile)
	require.Equal(t, nil, err)
	fd.Close()

	stat, err = exists(hiddenFile)
	require.Equal(t, nil, err)
	require.Equal(t, true, stat)

	os.Chmod(subdir, 0)

	stat, err = exists(hiddenFile)
	require.True(t, os.IsPermission(err))

	os.Chmod(subdir, 0700)
}
