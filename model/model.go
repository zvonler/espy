package model

import (
	"net/url"
	"time"
)

type Thread interface {
	URL() *url.URL
	Title() string
	Author() string
	StartDate() time.Time
	Latest() time.Time
	Replies() uint
	Views() uint
}

type Comment interface {
	Author() string
	Published() time.Time
	Content() string
}
