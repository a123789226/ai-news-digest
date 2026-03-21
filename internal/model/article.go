package model

import "time"

type Article struct {
	Source      string
	SourceType  string
	Title       string
	URL         string
	PublishedAt time.Time
	SummaryRaw  string
	ContentRaw  string
}

type DigestItem struct {
	TitleEN        string
	SummaryZH      string
	WhyItMattersZH string
	Source         string
	URL            string
}
