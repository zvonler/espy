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

type SiteID uint
type ForumID uint
type AuthorID uint
type ThreadID uint
type CommentID uint

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

func (sdb *ScraperDB) InsertOrUpdateForum(url *url.URL) (siteId SiteID, forumId ForumID, err error) {
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

func (sdb *ScraperDB) InsertOrUpdateThread(siteId SiteID, forumId ForumID, t model.Thread) (threadId ThreadID, err error) {
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

func (sdb *ScraperDB) GetSiteId(host string) (siteId SiteID, err error) {
	stmt := `SELECT id FROM site WHERE hostname = ?`
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) {
			err = rows.Scan(&siteId)
		},
		stmt, host)
	return
}

func (sdb *ScraperDB) GetThreadByURL(url *url.URL) (siteId SiteID, threadId ThreadID, err error) {
	stmt := `
		SELECT
			s.id, t.id
		FROM
			site s, forum f, thread t
		WHERE
			    s.id = f.site_id
			AND f.id = t.forum_id
			AND t.url = ?`

	err = errors.New("Not found") // rows.Scan will reset this if a row is found
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) {
			err = rows.Scan(&siteId, &threadId)
		},
		stmt, utils.TrimmedURL(url).String())
	return
}

func (sdb *ScraperDB) getOrInsertSite(hostname string) (id SiteID, err error) {
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

func (sdb *ScraperDB) getOrInsertAuthor(username string, siteId SiteID) (id AuthorID, err error) {
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

func (sdb *ScraperDB) AddComments(siteId SiteID, threadId ThreadID, comments []model.Comment) (err error) {
	for _, comment := range comments {
		var authorId AuthorID
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

func (sdb *ScraperDB) CommentTimeRange(threadId ThreadID) (res []time.Time) {
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

func (sdb *ScraperDB) FirstCommentLoaded(threadId ThreadID) (res bool) {
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

func (sdb *ScraperDB) SetForumLastScraped(forumId ForumID, time time.Time) {
	sdb.ExecOrPanic("UPDATE forum SET last_scraped = ? WHERE id = ?", time.Unix(), forumId)
}

func (sdb *ScraperDB) GetForumLastScraped(forumId ForumID) (tm time.Time, err error) {
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
	id integer not null primary key,
	url TEXT UNIQUE,
	thread_id INTEGER NOT NULL,
	author_id INTEGER NOT NULL,
	published INTEGER,
	content TEXT,

	UNIQUE(thread_id, author_id, published)
);
`
	_, err := sdb.DB.Exec(schema)
	if err != nil {
		log.Printf("Error loading schema: %q\n", err)
		return
	}
}
