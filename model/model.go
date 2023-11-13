package model

import (
	"net/url"
	"time"
)

type SiteID uint
type ForumID uint
type AuthorID uint
type ThreadID uint
type CommentID uint
type TagID uint

type Thread struct {
	Id        ThreadID
	URL       *url.URL
	Title     string
	Author    string
	StartDate time.Time
	Latest    time.Time
	Replies   uint
	Views     uint
}

type Comment struct {
	URL       *url.URL
	Author    string
	Published time.Time
	Content   string
}
