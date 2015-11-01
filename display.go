package main

import (
	"fmt"
	"html/template"
	"time"

	"github.com/gopherjs/gopherpen/issues"
	"github.com/shurcooL/htmlg"
)

// issueItem represents an event item for display purposes.
type issueItem struct {
	IssueItem
}

type IssueItem interface{}

func (i issueItem) TemplateName() string {
	switch i.IssueItem.(type) {
	case issues.Comment:
		return "comment"
	case issues.Event:
		return "event"
	default:
		panic("unknown item type")
	}
}

func (i issueItem) CreatedAt() time.Time {
	switch i := i.IssueItem.(type) {
	case issues.Comment:
		return i.CreatedAt
	case issues.Event:
		return i.CreatedAt
	default:
		panic("unknown item type")
	}
}

// byCreatedAt implements sort.Interface.
type byCreatedAt []issueItem

func (s byCreatedAt) Len() int           { return len(s) }
func (s byCreatedAt) Less(i, j int) bool { return s[i].CreatedAt().Before(s[j].CreatedAt()) }
func (s byCreatedAt) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// event is a issues.Event wrapper with display augmentations.
type event struct {
	issues.Event
}

func (e event) Text() template.HTML {
	switch e.Event.Type {
	case issues.Reopened, issues.Closed:
		return htmlg.Render(htmlg.Text(fmt.Sprintf("%s this", e.Event.Type)))
	case issues.Renamed:
		return htmlg.Render(htmlg.Text("changed the title from "), htmlg.Strong(e.Event.Rename.From), htmlg.Text(" to "), htmlg.Strong(e.Event.Rename.To))
	default:
		panic("unexpected event")
	}
}

func (e event) Octicon() string {
	switch e.Event.Type {
	case issues.Reopened:
		return "octicon-primitive-dot"
	case issues.Closed:
		return "octicon-circle-slash"
	case issues.Renamed:
		return "octicon-pencil"
	default:
		panic("unexpected event")
	}
}
