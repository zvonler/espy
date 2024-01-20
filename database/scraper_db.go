package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"time"

	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/zvonler/espy/model"
	"github.com/zvonler/espy/utils"
)

type ScraperDB struct {
	Filename string
	DB       *sql.DB
}

func regex(re, s string) (bool, error) {
	return regexp.MatchString(re, s)
}

func OpenScraperDB(path string) (sdb *ScraperDB, err error) {
	sql.Register("sqlite3_regex",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("regexp", regex, true)
			},
		})

	if existing_db, err := utils.PathExists(path); err == nil {
		if db, err := sql.Open("sqlite3_regex", path); err == nil {
			sdb = new(ScraperDB)
			sdb.Filename = path
			sdb.DB = db
			if !existing_db {
				sdb.initTables()
			}
		}
	}
	return
}

func (sdb *ScraperDB) Close() {
	sdb.DB.Close()
}

type RowsReceiver func(*sql.Rows)

func (sdb *ScraperDB) ForEachRowOrPanic(receiver RowsReceiver, stmt string, params ...any) {
	if rows, err := sdb.DB.Query(stmt, params...); err == nil {
		defer rows.Close()
		for rows.Next() {
			receiver(rows)
		}
	} else {
		panic(err)
	}
}

func (sdb *ScraperDB) ForSingleRowOrPanic(receiver RowsReceiver, stmt string, params ...any) {
	var rowReceived bool
	singleReceiver := func(rows *sql.Rows) {
		if rowReceived {
			panic(fmt.Sprintf("Received second row for %q", stmt))
		}
		receiver(rows)
		rowReceived = true
	}
	sdb.ForEachRowOrPanic(singleReceiver, stmt, params...)
}

func (sdb *ScraperDB) ExecOrPanic(stmt string, params ...any) {
	if _, err := sdb.DB.Exec(stmt, params...); err != nil {
		panic(err)
	}
}

func (sdb *ScraperDB) InsertOrUpdateForum(url *url.URL) (siteId model.SiteID, forumId model.ForumID, err error) {
	if siteId, err = sdb.getOrInsertSite(url.Hostname()); err == nil {
		sdb.ForSingleRowOrPanic(
			func(rows *sql.Rows) {
				err = rows.Scan(&forumId)
			},
			`INSERT INTO forum
				(site_id, url)
			VALUES
				(?, ?)
			ON CONFLICT
				DO UPDATE SET url = url
			RETURNING id`,
			siteId, utils.TrimmedURL(url).String())
	}
	return
}

func (sdb *ScraperDB) FindAuthorComments(username string) (comments []model.Comment, err error) {
	sdb.ForEachRowOrPanic(
		func(rows *sql.Rows) {
			var urlStr string
			var published int64
			var content string
			if err = rows.Scan(&urlStr, &published, &content); err == nil {
				if url, err := url.Parse(urlStr); err == nil {
					comments = append(comments, model.Comment{url, username, time.Unix(published, 0), content})
				}
			}
			if err != nil {
				panic(err)
			}
		},
		`SELECT
			url, published, content
		FROM comment c
		WHERE
			c.author_id IN (SELECT id FROM author WHERE username = ?)`,
		username)

	return
}

func (sdb *ScraperDB) InsertOrUpdateThread(siteId model.SiteID, forumId model.ForumID, t model.Thread) (threadId model.ThreadID, err error) {
	if authorId, err := sdb.getOrInsertAuthor(t.Author, siteId); err == nil {
		sdb.ForSingleRowOrPanic(
			func(rows *sql.Rows) {
				err = rows.Scan(&threadId)
			},
			`INSERT INTO thread
				(forum_id, author_id, title, url, replies, views, latest_activity, start_date)
			VALUES
				(?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT DO UPDATE SET
				replies = excluded.replies,
				views = excluded.views,
				latest_activity = excluded.latest_activity
			RETURNING id`,
			forumId, authorId, t.Title,
			utils.TrimmedURL(t.URL).String(), t.Replies,
			t.Views, t.Latest.Unix(), t.StartDate.Unix())
	}
	return
}

func (sdb *ScraperDB) GetSiteId(host string) (siteId model.SiteID, err error) {
	stmt := `SELECT id FROM site WHERE hostname = ?`
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) {
			err = rows.Scan(&siteId)
		},
		stmt, host)
	return
}

func (sdb *ScraperDB) GetThreadById(threadId model.ThreadID) (t model.Thread, err error) {
	stmt := `
		SELECT
			s.id, t.url, t.title, a.username, t.start_date, t.latest_activity, t.replies, t.views
		FROM
			site s, forum f, thread t, author a
		WHERE
				s.id = f.site_id
			AND f.id = t.forum_id
			AND a.id = t.author_id
			AND t.id = ?`

	err = errors.New("Not found") // rows.Scan will reset this if a row is found
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) {
			var urlStr string
			var startDate int64
			var latest int64
			err = rows.Scan(&t.SiteId, &urlStr, &t.Title, &t.Author, &startDate, &latest, t.Replies, &t.Views)
			t.StartDate = time.Unix(startDate, 0)
			t.Latest = time.Unix(latest, 0)
			t.Id = threadId
			t.URL, err = url.Parse(urlStr)
		},
		stmt, threadId)
	return
}

func (sdb *ScraperDB) GetThreadByURL(url *url.URL) (thread model.Thread, err error) {
	stmt := `
		SELECT
			s.id, t.id, t.title, a.username, t.start_date, t.latest_activity, t.replies, t.views
		FROM
			site s, forum f, thread t, author a
		WHERE
			    s.id = f.site_id
			AND f.id = t.forum_id
			AND a.id = t.author_id
			AND t.url = ?`

	err = errors.New("Not found") // rows.Scan will reset this if a row is found
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) {
			var startDate int64
			var latest int64
			err = rows.Scan(&thread.SiteId, &thread.Id, &thread.Title, &thread.Author, &startDate, &latest,
				&thread.Replies, &thread.Views)
			thread.StartDate = time.Unix(startDate, 0)
			thread.Latest = time.Unix(latest, 0)
			thread.URL = url
		},
		stmt, utils.TrimmedURL(url).String())
	return
}

func (sdb *ScraperDB) getOrInsertSite(hostname string) (id model.SiteID, err error) {
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) {
			err = rows.Scan(&id)
		},
		`INSERT INTO site
			(hostname)
		VALUES
			(?)
		ON CONFLICT DO UPDATE SET
			hostname = hostname
		RETURNING id`,
		hostname)
	return
}

func (sdb *ScraperDB) getOrInsertAuthor(username string, siteId model.SiteID) (id model.AuthorID, err error) {
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) {
			err = rows.Scan(&id)
		},
		`INSERT INTO author
			(site_id, username)
		VALUES
			(?, ?)
		ON CONFLICT DO UPDATE SET
			username = username
		RETURNING id`,
		siteId, username)
	return
}

func (sdb *ScraperDB) AddComments(siteId model.SiteID, threadId model.ThreadID, comments []model.Comment) (err error) {
	for _, comment := range comments {
		var authorId model.AuthorID
		if authorId, err = sdb.getOrInsertAuthor(comment.Author, siteId); err != nil {
			break
		}
		sdb.ExecOrPanic(
			`INSERT INTO comment
				(thread_id, url, author_id, published, content)
			VALUES
				(?, ?, ?, ?, ?)
			ON CONFLICT DO NOTHING`,
			threadId, comment.URL.String(), authorId, comment.Published.Unix(), comment.Content)
	}
	return
}

func (sdb *ScraperDB) GetForums() (forums []model.Forum, err error) {
	stmt := `SELECT id, url FROM forum ORDER BY id`

	sdb.ForEachRowOrPanic(
		func(rows *sql.Rows) {
			var id uint
			var urlStr string
			rows.Scan(&id, &urlStr)
			if url, err := url.Parse(urlStr); err == nil {
				forums = append(forums, model.Forum{model.ForumID(id), url})
			} else {
				panic(err)
			}
		},
		stmt)
	return
}

func (sdb *ScraperDB) GetSites() (hostnamesById map[model.SiteID]string, err error) {
	stmt := "SELECT id, hostname FROM site"
	hostnamesById = make(map[model.SiteID]string)
	sdb.ForEachRowOrPanic(
		func(rows *sql.Rows) {
			var id uint
			var hostname string
			rows.Scan(&id, &hostname)
			hostnamesById[model.SiteID(id)] = hostname
		},
		stmt)
	return
}

func (sdb *ScraperDB) GetThread(threadId model.ThreadID) (t model.Thread, err error) {
	if byId, err := sdb.GetThreads([]model.ThreadID{threadId}); err == nil {
		t = byId[threadId]
	}
	return
}

func (sdb *ScraperDB) GetThreads(threadIds []model.ThreadID) (threadsById map[model.ThreadID]model.Thread, err error) {
	stmt := `
		SELECT
			t.id, t.url, t.title, a.username, t.start_date, t.latest_activity, t.replies, t.views
		FROM thread t, author a
		WHERE
			a.id = t.author_id`

	threadsById = make(map[model.ThreadID]model.Thread)
	sdb.ForEachRowOrPanic(
		func(rows *sql.Rows) {
			var id uint
			var urlStr string
			var title string
			var username string
			var startDate int64
			var latest int64
			var replies uint
			var views uint

			rows.Scan(&id, &urlStr, &title, &username, &startDate, &latest, &replies, &views)
			if url, err := url.Parse(urlStr); err == nil {
				threadsById[model.ThreadID(id)] =
					model.Thread{
						URL:       url,
						Title:     title,
						Author:    username,
						StartDate: time.Unix(startDate, 0),
						Latest:    time.Unix(latest, 0),
						Replies:   replies,
						Views:     views,
					}
			} else {
				panic(err)
			}
		},
		stmt)
	return
}

// Finds a thread in the database by either URL or ID.
func (sdb *ScraperDB) FindThread(arg string) (thread model.Thread, err error) {
	if url, id, err := utils.ParseURLOrID(arg); err == nil {
		if url != nil {
			thread, err = sdb.GetThreadByURL(url)
		} else {
			thread, err = sdb.GetThreadById(model.ThreadID(id))
		}
	}
	return
}

func (sdb *ScraperDB) ThreadComments(threadId model.ThreadID) (comments []model.Comment, err error) {
	stmt := `
		SELECT
			c.url, a.username, c.published, c.content
		FROM author a, comment c, thread t
		WHERE
				a.id = c.author_id
			AND c.thread_id = t.id
			AND t.id = ?
		ORDER BY published`

	sdb.ForEachRowOrPanic(
		func(rows *sql.Rows) {
			var urlStr string
			var username string
			var published uint
			var content string
			err = rows.Scan(&urlStr, &username, &published, &content)
			if err != nil {
				panic(err)
			}
			url, _ := url.Parse(urlStr)
			comments = append(comments,
				model.Comment{
					URL:       url,
					Author:    username,
					Published: time.Unix(int64(published), 0),
					Content:   content,
				})
		}, stmt, threadId)

	return
}

func (sdb *ScraperDB) getOrInsertTagId(tag string) (id model.TagID, err error) {
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) {
			err = rows.Scan(&id)
		},
		`INSERT INTO tag
			(name)
		VALUES
			(?)
		ON CONFLICT DO UPDATE SET
			name = name
		RETURNING id`,
		tag)
	return
}

func (sdb *ScraperDB) AddThreadTags(threadId model.ThreadID, tags []string) (err error) {
	for _, tag := range tags {
		if tagId, err := sdb.getOrInsertTagId(tag); err == nil {
			stmt := `
				INSERT INTO thread_tag
					(thread_id, tag_id)
				VALUES
					(?, ?)
				ON CONFLICT DO NOTHING`
			sdb.ExecOrPanic(stmt, threadId, tagId)
		} else {
			break
		}
	}
	return
}

func (sdb *ScraperDB) RemoveThreadTags(threadId model.ThreadID, tags []string) (err error) {
	for _, tag := range tags {
		if tagId, err := sdb.getOrInsertTagId(tag); err == nil {
			stmt := "DELETE FROM thread_tag WHERE thread_id = ? AND tag_id = ?"
			sdb.ExecOrPanic(stmt, threadId, tagId)
		} else {
			break
		}
	}
	return
}

func (sdb *ScraperDB) ThreadParticipants(threadId model.ThreadID) (usernames []string, err error) {
	stmt := `
		SELECT DISTINCT
			username
		FROM author a, comment c, thread t
		WHERE
				a.id = c.author_id
			AND c.thread_id = t.id
			AND t.id = ?
		ORDER BY published`

	usernameSet := make(map[string]bool)

	sdb.ForEachRowOrPanic(
		func(rows *sql.Rows) {
			var username string
			err = rows.Scan(&username)
			if err != nil {
				panic(err)
			}
			usernameSet[username] = true
		}, stmt, threadId)

	usernames = make([]string, len(usernameSet))
	i := 0
	for username, _ := range usernameSet {
		usernames[i] = username
		i++
	}
	return
}

func (sdb *ScraperDB) CommentTimeRange(threadId model.ThreadID) (res []time.Time) {
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) {
			var earliest, latest uint
			if err := rows.Scan(&earliest, &latest); err == nil {
				res = []time.Time{time.Unix(int64(earliest), 0), time.Unix(int64(latest), 0)}
			}
		},
		`SELECT MIN(published), MAX(published) FROM COMMENT WHERE thread_id = ?`,
		threadId)
	return
}

func (sdb *ScraperDB) FirstCommentLoaded(threadId model.ThreadID) (res bool) {
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) {
			res = true
		},
		`SELECT MIN(published) FROM comment WHERE thread_id = ?
			INTERSECT
		SELECT start_date FROM thread WHERE id = ?`,
		threadId, threadId)
	return
}

func (sdb *ScraperDB) SetForumLastScraped(forumId model.ForumID, time time.Time) {
	sdb.ExecOrPanic("UPDATE forum SET last_scraped = ? WHERE id = ?", time.Unix(), forumId)
}

func (sdb *ScraperDB) GetForumLastScraped(forumId model.ForumID) (tm time.Time, err error) {
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) {
			var epochSecs int64
			err = rows.Scan(&epochSecs)
			tm = time.Unix(epochSecs, 0)
		},
		"SELECT last_scraped FROM forum WHERE id = ?",
		forumId)
	return
}

func (sdb *ScraperDB) initTables() {
	schema := `
CREATE TABLE site (
	id INTEGER NOT NULL PRIMARY KEY,
	hostname STRING UNIQUE
);

CREATE TABLE forum (
	id INTEGER NOT NULL PRIMARY KEY,
	site_id INTEGER NOT NULL,
	url TEXT UNIQUE,
	last_scraped INTEGER
);

CREATE TABLE author (
	id INTEGER NOT NULL PRIMARY KEY,
	site_id INTEGER NOT NULL,
	username TEXT,

	UNIQUE(site_id, username)
);

CREATE TABLE thread (
	id INTEGER NOT NULL PRIMARY KEY,
	forum_id INTEGER NOT NULL,
	author_id INTEGER NOT NULL,
	title TEXT,
	url TEXT UNIQUE,
	replies INTEGER,
	views INTEGER,
	latest_activity INTEGER,
	start_date INTEGER
);

CREATE TABLE comment (
	id INTEGER NOT NULL PRIMARY KEY,
	url TEXT UNIQUE,
	thread_id INTEGER NOT NULL,
	author_id INTEGER NOT NULL,
	published INTEGER,
	content TEXT,

	UNIQUE(thread_id, author_id, published)
);

CREATE TABLE thread_tag (
	thread_id INTEGER NOT NULL,
	tag_id INTEGER NOT NULL,

	UNIQUE(thread_id, tag_id)
);

CREATE TABLE tag (
	id INTEGER NOT NULL PRIMARY KEY,
	name TEXT UNIQUE
);
`
	_, err := sdb.DB.Exec(schema)
	if err != nil {
		log.Printf("Error loading schema: %q\n", err)
		return
	}
}
