// Package component contains individual components that can render themselves as HTML.
package component

import (
	"image/color"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/octiconssvg"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// IssueStateBadge is a component that displays the state of an issue
// with a badge, who opened it, and when it was opened.
type IssueStateBadge struct {
	Issue issues.Issue
}

func (i IssueStateBadge) Render() []*html.Node {
	// TODO: Make this much nicer.
	// {{render (issueBadge .State)}}
	// <span style="margin-left: 4px;">{{render (user .User)}} opened this issue {{render (time .CreatedAt)}}</span>
	var ns []*html.Node
	ns = append(ns, IssueBadge{State: i.Issue.State}.Render()...)
	span := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{
			{Key: atom.Style.String(), Val: "margin-left: 4px;"},
		},
	}
	for _, n := range (User{i.Issue.User}).Render() {
		span.AppendChild(n)
	}
	span.AppendChild(htmlg.Text(" opened this issue "))
	for _, n := range (Time{i.Issue.CreatedAt}).Render() {
		span.AppendChild(n)
	}
	ns = append(ns, span)
	return ns
}

// IssueBadge is an issue badge, displaying the issue's state.
type IssueBadge struct {
	State issues.State
}

func (ib IssueBadge) Render() []*html.Node {
	// TODO: Make this much nicer.
	// {{if eq . "open"}}
	// 	<span style="display: inline-block; padding: 4px 6px 4px 6px; margin: 4px; color: #fff; background-color: #6cc644;"><span style="margin-right: 6px;" class="octicon octicon-issue-opened"></span>Open</span>
	// {{else if eq . "closed"}}
	// 	<span style="display: inline-block; padding: 4px 6px 4px 6px; margin: 4px; color: #fff; background-color: #bd2c00;"><span style="margin-right: 6px;" class="octicon octicon-issue-closed"></span>Closed</span>
	// {{else}}
	// 	{{.}}
	// {{end}}
	var (
		icon  *html.Node
		text  string
		color string
	)
	switch ib.State {
	case issues.OpenState:
		icon = octiconssvg.IssueOpened()
		text = "Open"
		color = "#6cc644"
	case issues.ClosedState:
		icon = octiconssvg.IssueClosed()
		text = "Closed"
		color = "#bd2c00"
	default:
		return []*html.Node{htmlg.Text(string(ib.State))}
	}
	span := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{{
			Key: atom.Style.String(),
			Val: `display: inline-block;
padding: 4px 6px 4px 6px;
margin: 4px;
color: #fff;
background-color: ` + color + `;`,
		}},
	}
	span.AppendChild(&html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr:       []html.Attribute{{Key: atom.Style.String(), Val: "margin-right: 6px;"}},
		FirstChild: icon,
	})
	span.AppendChild(htmlg.Text(text))
	return []*html.Node{span}
}

// IssueIcon is an issue icon, displaying the issue's state.
type IssueIcon struct {
	State issues.State
}

func (ii IssueIcon) Render() []*html.Node {
	// TODO: Make this much nicer.
	// {{if eq . "open"}}
	// 	<span style="margin-right: 6px; color: #6cc644;" class="octicon octicon-issue-opened"></span>
	// {{else if eq . "closed"}}
	// 	<span style="margin-right: 6px; color: #bd2c00;" class="octicon octicon-issue-closed"></span>
	// {{end}}
	var (
		icon  *html.Node
		color string
	)
	switch ii.State {
	case issues.OpenState:
		icon = octiconssvg.IssueOpened()
		color = "#6cc644"
	case issues.ClosedState:
		icon = octiconssvg.IssueClosed()
		color = "#bd2c00"
	}
	span := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{{
			Key: atom.Style.String(),
			Val: `margin-right: 6px;
color: ` + color + `;`,
		}},
		FirstChild: icon,
	}
	return []*html.Node{span}
}

// Label is a label component.
type Label struct {
	Label issues.Label
}

func (l Label) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <span style="...; color: {{.fontColor}}; background-color: {{.Color.HexString}};">{{.Name}}</span>
	span := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{{
			Key: atom.Style.String(),
			Val: `display: inline-block;
font-size: 12px;
line-height: 1.2;
padding: 0px 3px 0px 3px;
border-radius: 2px;
color: ` + l.fontColor() + `;
background-color: ` + l.Label.Color.HexString() + `;`,
		}},
	}
	span.AppendChild(htmlg.Text(l.Label.Name))
	return []*html.Node{span}
}

// fontColor returns one of "#fff" or "#000", whichever is a better fit for
// the font color given the label color.
func (l Label) fontColor() string {
	// Convert label color to 8-bit grayscale, and make a decision based on that.
	switch y := color.GrayModel.Convert(l.Label.Color).(color.Gray).Y; {
	case y < 128:
		return "#fff"
	case y >= 128:
		return "#000"
	}
	panic("unreachable")
}

// User is a user component.
type User struct {
	User users.User
}

func (u User) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <a class="black" href="{{.HTMLURL}}" target="_blank"><strong>{{.Login}}</strong></a>
	a := &html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Class.String(), Val: "black"},
			{Key: atom.Href.String(), Val: u.User.HTMLURL},
			{Key: atom.Target.String(), Val: "_blank"},
		},
		FirstChild: htmlg.Strong(u.User.Login),
	}
	return []*html.Node{a}
}

// Time component that displays human friendly relative time (e.g., "2 hours ago", "yesterday"),
// but also contains a tooltip with the full absolute time (e.g., "Jan 2, 2006, 3:04 PM MST").
//
// TODO: Factor out, it's the same as in notificationsapp.
type Time struct {
	Time time.Time
}

func (t Time) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <abbr title="{{.Format "Jan 2, 2006, 3:04 PM MST"}}">{{reltime .}}</abbr>
	abbr := &html.Node{
		Type: html.ElementNode, Data: atom.Abbr.String(),
		Attr:       []html.Attribute{{Key: atom.Title.String(), Val: t.Time.Format("Jan 2, 2006, 3:04 PM MST")}},
		FirstChild: htmlg.Text(humanize.Time(t.Time)),
	}
	return []*html.Node{abbr}
}
