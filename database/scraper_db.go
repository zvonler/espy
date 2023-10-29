package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
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
	Filename             string
	DB                   *sql.DB
	insertForumStmt      string
	insertThreadStmt     string
	insertSiteStmt       string
	insertAuthorStmt     string
	insertCommentStmt    string
	commentTimeRangeStmt string
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

	if existing_db, err := exists(path); err == nil {
		if db, err := sql.Open("sqlite3_regex", path); err == nil {
			sdb = new(ScraperDB)
			sdb.Filename = path
			sdb.DB = db
			if !existing_db {
				sdb.initTables()
			}
			sdb.initSQLStatements()
		}
	}
	return
}

func (sdb *ScraperDB) Close() {
	sdb.DB.Close()
}

type RowsReceiver func(*sql.Rows) bool

func (sdb *ScraperDB) ForEachRowOrPanic(receiver RowsReceiver, stmt string, params ...any) {
	if rows, err := sdb.DB.Query(stmt, params...); err == nil {
		defer rows.Close()
		for rows.Next() {
			if !receiver(rows) {
				break
			}
		}
	} else {
		panic(err)
	}
}

func (sdb *ScraperDB) ForSingleRowOrPanic(receiver RowsReceiver, stmt string, params ...any) {
	var rowReceived bool
	singleReceiver := func(rows *sql.Rows) bool {
		if rowReceived {
			panic(fmt.Sprintf("Received second row for %q", stmt))
		}
		receiver(rows)
		rowReceived = true
		return true
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
			func(rows *sql.Rows) bool {
				err = rows.Scan(&forumId)
				return true
			},
			sdb.insertForumStmt, siteId, utils.TrimmedURL(url).String())
	}
	return
}

func (sdb *ScraperDB) InsertOrUpdateThread(siteId SiteID, forumId ForumID, t model.Thread) (threadId ThreadID, err error) {
	if authorId, err := sdb.getOrInsertAuthor(t.Author, siteId); err == nil {
		sdb.ForSingleRowOrPanic(
			func(rows *sql.Rows) bool {
				err = rows.Scan(&threadId)
				return true
			},
			sdb.insertThreadStmt,
			forumId, authorId, t.Title,
			utils.TrimmedURL(t.URL).String(), t.Replies,
			t.Views, t.Latest.Unix(), t.StartDate.Unix())
	}
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

	var rows *sql.Rows
	if rows, err = sdb.DB.Query(stmt, utils.TrimmedURL(url).String()); err == nil {
		defer rows.Close()
		if rows.Next() {
			err = rows.Scan(&siteId, &threadId)
		} else {
			err = errors.New("Not found")
		}
	}
	return
}

func (sdb *ScraperDB) getOrInsertSite(hostname string) (id SiteID, err error) {
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) bool {
			err = rows.Scan(&id)
			return true
		},
		sdb.insertSiteStmt, hostname)
	return
}

func (sdb *ScraperDB) getOrInsertAuthor(username string, siteId SiteID) (id AuthorID, err error) {
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) bool {
			err = rows.Scan(&id)
			return true
		},
		sdb.insertAuthorStmt, siteId, username)
	return
}

func (sdb *ScraperDB) AddComments(siteId SiteID, threadId ThreadID, comments []model.Comment) {
	if len(comments) == 0 {
		return
	}

	for _, comment := range comments {
		authorId, err := sdb.getOrInsertAuthor(comment.Author, siteId)
		if err != nil {
			log.Fatal(err)
		}
		sdb.ExecOrPanic(sdb.insertCommentStmt, threadId, authorId, comment.Published.Unix(), comment.Content)
	}
}

func (sdb *ScraperDB) CommentTimeRange(threadId ThreadID) (res []time.Time) {
	sdb.ForSingleRowOrPanic(
		func(rows *sql.Rows) bool {
			var earliest, latest uint
			if err := rows.Scan(&earliest, &latest); err == nil {
				res = []time.Time{time.Unix(int64(earliest), 0), time.Unix(int64(latest), 0)}
			}
			return true
		},
		sdb.commentTimeRangeStmt, threadId)
	return
}

func (sdb *ScraperDB) FirstCommentLoaded(threadId ThreadID) bool {
	sql := `
		SELECT MIN(published) FROM comment WHERE thread_id = ?
			INTERSECT
		SELECT start_date FROM thread WHERE id = ?`
	rows, err := sdb.DB.Query(sql, threadId, threadId)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	return rows.Next()
}

func (sdb *ScraperDB) SetForumLastScraped(forumId ForumID, time time.Time) {
	sdb.ExecOrPanic("UPDATE forum SET last_scraped = ? WHERE id = ?", time.Unix(), forumId)
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

func (sdb *ScraperDB) initSQLStatements() {
	sdb.insertForumStmt = `
		INSERT INTO forum
			(site_id, url)
		VALUES
			(?, ?)
		ON CONFLICT
			DO UPDATE SET url = url
		RETURNING id`

	sdb.insertThreadStmt = `
		INSERT INTO thread
			(forum_id, author_id, title, url, replies, views, latest_activity, start_date)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT DO UPDATE SET
			replies = excluded.replies,
			views = excluded.views,
			latest_activity = excluded.latest_activity
		RETURNING id`

	sdb.insertSiteStmt = `
		INSERT INTO site
			(hostname)
		VALUES
			(?)
		ON CONFLICT DO UPDATE SET
			hostname = hostname
		RETURNING id`

	sdb.insertAuthorStmt = `
		INSERT INTO author
			(site_id, username)
		VALUES
			(?, ?)
		ON CONFLICT DO UPDATE SET
			username = username
		RETURNING id`

	sdb.insertCommentStmt = `
		INSERT INTO comment
			(thread_id, author_id, published, content)
		VALUES
			(?, ?, ?, ?)
		ON CONFLICT DO NOTHING`

	sdb.commentTimeRangeStmt = `
		SELECT MIN(published), MAX(published) FROM COMMENT WHERE thread_id = ?`
}

func exists(path string) (res bool, err error) {
	_, statErr := os.Stat(path)
	if statErr == nil {
		res = true
	} else if !os.IsNotExist(statErr) {
		err = statErr
	}
	return
}
