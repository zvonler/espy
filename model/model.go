package model

import (
	"net/url"
	"time"
)

type Thread struct {
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
