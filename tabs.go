package issuesapp

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"

	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/octiconssvg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// TODO: Factor out somehow...
var tabsTmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"octicon": func(name string) (template.HTML, error) {
		icon := octiconssvg.Icon(name)
		if icon == nil {
			return "", fmt.Errorf("%q is not a valid Octicon symbol name", name)
		}
		var buf bytes.Buffer
		err := html.Render(&buf, icon)
		if err != nil {
			return "", err
		}
		return template.HTML(buf.String()), nil
	},
}).Parse(`
{{define "open-issue-count"}}<span><span style="margin-right: 4px;">{{octicon "issue-opened"}}</span>{{.OpenCount}} Open</span>{{end}}
{{define "closed-issue-count"}}<span style="margin-left: 12px;"><span style="margin-right: 4px;">{{octicon "check"}}</span>{{.ClosedCount}} Closed</span>{{end}}
`))

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
		id           string
		templateName string
	}{
		{id: "", templateName: "open-issue-count"},
		{id: string(issues.ClosedState), templateName: "closed-issue-count"},
	} {
		a := &html.Node{Type: html.ElementNode, Data: atom.A.String()}
		if tab.id == selectedTab {
			a.Attr = []html.Attribute{{Key: atom.Class.String(), Val: "selected"}}
		} else {
			q := query
			if tab.id == "" {
				q.Del(queryKeyState)
			} else {
				q.Set(queryKeyState, tab.id)
			}
			u := url.URL{
				Path:     path,
				RawQuery: q.Encode(),
			}
			a.Attr = []html.Attribute{
				{Key: atom.Href.String(), Val: u.String()},
			}
		}
		// TODO: This is horribly inefficient... :o
		var buf bytes.Buffer
		err := tabsTmpl.ExecuteTemplate(&buf, tab.templateName, s)
		if err != nil {
			return "", err
		}
		tmplNode, err := html.Parse(&buf)
		if err != nil {
			return "", err
		}
		a.AppendChild(tmplNode)
		ns = append(ns, a)
	}

	return template.HTML(htmlg.Render(ns...)), nil
}
