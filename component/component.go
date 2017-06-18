// Package component contains individual components that can render themselves as HTML.
package component

import (
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/octiconssvg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

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
