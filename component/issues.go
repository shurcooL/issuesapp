package component

import (
	"fmt"

	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/octiconssvg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// IssueEntry is an entry within the list of issues.
type IssueEntry struct {
	Issue  issues.Issue
	Unread bool // Unread indicates whether the issue contains unread notifications for authenticated user.

	// TODO, THINK: This is router details, can it be factored out or cleaned up?
	BaseURI string
}

func (i IssueEntry) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <div class="list-entry-body multilist-entry"{{if .Unread}} style="box-shadow: 2px 0 0 #4183c4 inset;"{{end}}>
	// 	<div style="display: flex;">
	// 		{{render (issueIcon .State)}}
	// 		<div style="flex-grow: 1;">
	// 			<div>
	// 				<a class="black" href="{{state.BaseURI}}/{{.ID}}"><strong>{{.Title}}</strong></a>
	// 				{{range .Labels}}{{render (label .)}}{{end}}
	// 			</div>
	// 			<div class="gray tiny">#{{.ID}} opened {{render (time .CreatedAt)}} by {{.User.Login}}</div>
	// 		</div>
	// 		<span title="{{.Replies}} replies" class="tiny {{if .Replies}}gray{{else}}lightgray{{end}}">{{octicon "comment"}} {{.Replies}}</span>
	// 	</div>
	// </div>

	div := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "display: flex;"}},
	}
	htmlg.AppendChildren(div, IssueIcon{State: i.Issue.State}.Render()...)

	titleAndByline := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "flex-grow: 1;"}},
	}
	{
		title := htmlg.Div(
			&html.Node{
				Type: html.ElementNode, Data: atom.A.String(),
				Attr: []html.Attribute{
					{Key: atom.Class.String(), Val: "black"},
					{Key: atom.Href.String(), Val: fmt.Sprintf("%s/%d", i.BaseURI, i.Issue.ID)},
				},
				FirstChild: htmlg.Strong(i.Issue.Title),
			},
		)
		for _, l := range i.Issue.Labels {
			span := &html.Node{
				Type: html.ElementNode, Data: atom.Span.String(),
				Attr: []html.Attribute{{Key: atom.Style.String(), Val: "margin-left: 4px;"}},
			}
			htmlg.AppendChildren(span, Label{Label: l}.Render()...)
			title.AppendChild(span)
		}
		titleAndByline.AppendChild(title)

		byline := htmlg.DivClass("gray tiny")
		byline.AppendChild(htmlg.Text(fmt.Sprintf("#%d opened ", i.Issue.ID)))
		htmlg.AppendChildren(byline, Time{Time: i.Issue.CreatedAt}.Render()...)
		byline.AppendChild(htmlg.Text(fmt.Sprintf(" by %s", i.Issue.User.Login)))
		titleAndByline.AppendChild(byline)
	}
	div.AppendChild(titleAndByline)

	spanClass := "tiny"
	switch i.Issue.Replies {
	default:
		spanClass += " gray"
	case 0:
		spanClass += " lightgray"
	}
	span := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{
			{Key: atom.Title.String(), Val: fmt.Sprintf("%d replies", i.Issue.Replies)},
			{Key: atom.Class.String(), Val: spanClass},
		},
	}
	span.AppendChild(octiconssvg.Comment())
	span.AppendChild(htmlg.Text(fmt.Sprintf(" %d", i.Issue.Replies)))
	div.AppendChild(span)

	listEntryDiv := htmlg.DivClass("list-entry-body multilist-entry", div)
	if i.Unread {
		listEntryDiv.Attr = append(listEntryDiv.Attr,
			html.Attribute{Key: atom.Style.String(), Val: "box-shadow: 2px 0 0 #4183c4 inset;"},
		)
	}
	return []*html.Node{listEntryDiv}
}
