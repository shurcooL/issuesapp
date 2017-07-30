package issuesapp

import (
	"fmt"
	"html/template"
	"net/url"

	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/issuesapp/component"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	queryKeyState = "state"
)

// TODO: Reorganize and deduplicate.
func tab(query url.Values) (issues.State, error) {
	switch query.Get(queryKeyState) {
	case "":
		return issues.OpenState, nil
	case string(issues.ClosedState):
		return issues.ClosedState, nil
	default:
		return "", fmt.Errorf("unsupported queryKeyState value: %q", query.Get(queryKeyState))
	}
}

// tabs renders the html for <nav> element with tab header links.
func tabs(s *state, path string, rawQuery string) (template.HTML, error) {
	query, _ := url.ParseQuery(rawQuery)

	selectedTab := query.Get(queryKeyState)

	var ns []*html.Node

	for _, tab := range []struct {
		ID        string
		Component htmlg.Component
	}{
		{ID: "", Component: &component.OpenIssuesTab{}},
		{ID: string(issues.ClosedState), Component: &component.ClosedIssuesTab{}},
	} {
		a := &html.Node{Type: html.ElementNode, Data: atom.A.String()}
		if tab.ID == selectedTab {
			a.Attr = []html.Attribute{{Key: atom.Class.String(), Val: "selected"}}
		} else {
			q := query
			if tab.ID == "" {
				q.Del(queryKeyState)
			} else {
				q.Set(queryKeyState, tab.ID)
			}
			u := url.URL{
				Path:     path,
				RawQuery: q.Encode(),
			}
			a.Attr = []html.Attribute{
				{Key: atom.Href.String(), Val: u.String()},
			}
		}
		switch t := tab.Component.(type) {
		case *component.OpenIssuesTab:
			openCount, err := s.is.Count(s.req.Context(), s.RepoSpec, issues.IssueListOptions{State: issues.StateFilter(issues.OpenState)})
			if err != nil {
				return "", err
			}
			t.Count = openCount
		case *component.ClosedIssuesTab:
			closedCount, err := s.is.Count(s.req.Context(), s.RepoSpec, issues.IssueListOptions{State: issues.StateFilter(issues.ClosedState)})
			if err != nil {
				return "", err
			}
			t.Count = closedCount
		}
		for _, n := range tab.Component.Render() {
			a.AppendChild(n)
		}
		ns = append(ns, a)
	}

	return template.HTML(htmlg.Render(ns...)), nil
}
