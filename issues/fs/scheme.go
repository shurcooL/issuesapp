package fs

import "time"

// issue is an on-disk representation of an issue.
type issue struct {
	State string
	Title string
	comment
}

// comment is an on-disk representation of a comment.
type comment struct {
	AuthorUID int32
	CreatedAt time.Time
	Body      string
}
