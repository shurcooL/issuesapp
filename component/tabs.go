package component

import (
	"fmt"

	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/octiconssvg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// OpenIssuesTab is an "Open Issues Tab" component.
type OpenIssuesTab struct {
	Count uint64 // Count of open issues.
}

func (t OpenIssuesTab) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <span>
	// 	<span style="margin-right: 4px;">{{octicon "issue-opened"}}</span>
	// 	{{.Count}} Open
	// </span>
	iconSpan := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{
			{Key: atom.Style.String(), Val: "margin-right: 4px;"},
		},
		FirstChild: octiconssvg.IssueOpened(),
	}
	text := htmlg.Text(fmt.Sprintf("%d Open", t.Count))
	span := htmlg.Span(iconSpan, text)
	return []*html.Node{span}
}

// ClosedIssuesTab is a "Closed Issues Tab" component.
type ClosedIssuesTab struct {
	Count uint64 // Count of closed issues.
}

func (t ClosedIssuesTab) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <span style="margin-left: 12px;">
	// 	<span style="margin-right: 4px;">{{octicon "check"}}</span>
	// 	{{.Count}} Closed
	// </span>
	iconSpan := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{
			{Key: atom.Style.String(), Val: "margin-right: 4px;"},
		},
		FirstChild: octiconssvg.Check(),
	}
	text := htmlg.Text(fmt.Sprintf("%d Closed", t.Count))
	span := htmlg.Span(iconSpan, text)
	span.Attr = []html.Attribute{
		{Key: atom.Style.String(), Val: "margin-left: 12px;"},
	}
	return []*html.Node{span}
}
