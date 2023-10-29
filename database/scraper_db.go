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
	insertForumStmt      *sql.Stmt
	insertThreadStmt     *sql.Stmt
	insertSiteStmt       *sql.Stmt
	insertAuthorStmt     *sql.Stmt
	insertCommentStmt    *sql.Stmt
	commentTimeRangeStmt *sql.Stmt
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

func (sdb *ScraperDB) InsertOrUpdateForum(url *url.URL) (siteId SiteID, forumId ForumID) {
	tx, err := sdb.DB.Begin()
	if err != nil {
		panic(err)
	}

	siteId = sdb.getOrInsertSite(url.Hostname(), tx)

	rows, err := tx.Stmt(sdb.insertForumStmt).Query(siteId, utils.TrimmedURL(url).String())
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&forumId)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		panic("No return from insert")
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	return
}

func (sdb *ScraperDB) InsertOrUpdateThread(siteId SiteID, forumId ForumID, t model.Thread) (threadId ThreadID) {
	tx, err := sdb.DB.Begin()
	if err != nil {
		log.Fatal(err)
	}
	authorId := sdb.getOrInsertAuthor(t.Author, siteId, tx)

	stmt := tx.Stmt(sdb.insertThreadStmt)
	rows, err := stmt.Query(forumId, authorId, t.Title, utils.TrimmedURL(t.URL).String(),
		t.Replies, t.Views, t.Latest.Unix(), t.StartDate.Unix())
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&threadId)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		panic("No return from insert")
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
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

func (sdb *ScraperDB) getOrInsertSite(hostname string, tx *sql.Tx) (id SiteID) {
	if rows, err := tx.Stmt(sdb.insertSiteStmt).Query(hostname); err == nil {
		defer rows.Close()
		if rows.Next() {
			err = rows.Scan(&id)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal("No return from insert")
		}
	} else {
		log.Fatal(err)
	}
	return
}

func (sdb *ScraperDB) getOrInsertAuthor(username string, siteId SiteID, tx *sql.Tx) (id AuthorID) {
	rows, err := tx.Stmt(sdb.insertAuthorStmt).Query(siteId, username)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		panic("No return from insert")
	}
	return
}

func (sdb *ScraperDB) AddComments(siteId SiteID, threadId ThreadID, comments []model.Comment) {
	if len(comments) == 0 {
		return
	}

	tx, err := sdb.DB.Begin()
	if err != nil {
		log.Fatal(err)
	}

	for _, comment := range comments {
		authorId := sdb.getOrInsertAuthor(comment.Author, siteId, tx)
		_, err := tx.Stmt(sdb.insertCommentStmt).Exec(threadId, authorId, comment.Published.Unix(), comment.Content)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

func (sdb *ScraperDB) CommentTimeRange(threadId ThreadID) []time.Time {
	rows, err := sdb.commentTimeRangeStmt.Query(threadId)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		var earliest, latest uint
		err := rows.Scan(&earliest, &latest)
		if err == nil {
			return []time.Time{time.Unix(int64(earliest), 0), time.Unix(int64(latest), 0)}
		}
	}
	return nil
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
	var err error

	sdb.insertForumStmt, err = sdb.DB.Prepare(`
		INSERT INTO forum
			(site_id, url)
		VALUES
			(?, ?)
		ON CONFLICT
			DO UPDATE SET url = url
		RETURNING id`)
	if err != nil {
		log.Fatal(err)
	}

	sdb.insertThreadStmt, err = sdb.DB.Prepare(`
		INSERT INTO thread
			(forum_id, author_id, title, url, replies, views, latest_activity, start_date)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT DO UPDATE SET
			replies = excluded.replies,
			views = excluded.views,
			latest_activity = excluded.latest_activity
		RETURNING id`)
	if err != nil {
		log.Fatal(err)
	}

	sdb.insertSiteStmt, err = sdb.DB.Prepare(`
		INSERT INTO site
			(hostname)
		VALUES
			(?)
		ON CONFLICT DO UPDATE SET
			hostname = hostname
		RETURNING id`)
	if err != nil {
		log.Fatal(err)
	}

	sdb.insertAuthorStmt, err = sdb.DB.Prepare(`
		INSERT INTO author
			(site_id, username)
		VALUES
			(?, ?)
		ON CONFLICT DO UPDATE SET
			username = username
		RETURNING id`)
	if err != nil {
		log.Fatal(err)
	}

	sdb.insertCommentStmt, err = sdb.DB.Prepare(`
		INSERT INTO comment
			(thread_id, author_id, published, content)
		VALUES
			(?, ?, ?, ?)
		ON CONFLICT DO NOTHING`)
	if err != nil {
		log.Fatal(err)
	}

	sdb.commentTimeRangeStmt, err = sdb.DB.Prepare(`
		SELECT MIN(published), MAX(published) FROM COMMENT WHERE thread_id = ?`)
	if err != nil {
		log.Fatal(err)
	}
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
