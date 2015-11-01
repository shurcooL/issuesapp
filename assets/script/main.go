// +build js

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/gopherpen/common"
	"github.com/gopherjs/gopherpen/issues"
	"github.com/shurcooL/github_flavored_markdown"
	"honnef.co/go/js/dom"
)

var document = dom.GetWindow().Document().(dom.HTMLDocument)

var state common.State

func main() {
	js.Global.Set("MarkdownPreview", MarkdownPreview)
	js.Global.Set("SwitchWriteTab", SwitchWriteTab)
	js.Global.Set("CreateNewIssue", CreateNewIssue)
	js.Global.Set("ToggleIssueState", ToggleIssueState)
	js.Global.Set("PostComment", PostComment)

	stateJSON := js.Global.Get("State").String()
	fmt.Println(stateJSON)
	err := json.Unmarshal([]byte(stateJSON), &state)
	if err != nil {
		panic(err)
	}
}

func CreateNewIssue() {
	titleEditor := document.GetElementByID("title-editor").(*dom.HTMLInputElement)
	commentEditor := document.GetElementByID("comment-editor").(*dom.HTMLTextAreaElement)

	title := titleEditor.Value
	body := commentEditor.Value
	if strings.TrimSpace(title) == "" {
		log.Println("cannot create issue with empty title")
		return
	}

	go func() {
		resp, err := http.PostForm("new", url.Values{"csrf_token": {state.CSRFToken}, "title": {title}, "body": {body}})
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Printf("got reply: %v\n%q\n", resp.Status, string(body))

		switch resp.StatusCode {
		case http.StatusOK:
			// TODO: Redirect?
		}
	}()
}

func ToggleIssueState(issueState string) {
	newString := func(s string) *string { return &s }

	ir := issues.IssueRequest{
		State: newString(issueState),
	}
	value, err := json.Marshal(ir)
	if err != nil {
		panic(err)
	}

	go func() {
		resp, err := http.PostForm(state.BaseURI+state.ReqPath+"/edit", url.Values{"csrf_token": {state.CSRFToken}, "value": {string(value)}})
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return
		}

		data, err := url.ParseQuery(string(body))
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Printf("got reply: %v\n%q\n", resp.Status, data)

		switch resp.StatusCode {
		case http.StatusOK:
			issueStateBadge := document.GetElementByID("issue-state-badge")
			issueStateBadge.SetInnerHTML(data.Get("issue-state-badge"))

			issueToggleButton := document.GetElementByID("issue-toggle-button")
			issueToggleButton.Underlying().Set("outerHTML", data.Get("issue-toggle-button"))

			// Create event.
			newEvent := document.CreateElement("div").(*dom.HTMLDivElement)
			newItemMarker := document.GetElementByID("new-item-marker")
			newItemMarker.ParentNode().InsertBefore(newEvent, newItemMarker)
			newEvent.Underlying().Set("outerHTML", data.Get("new-event"))
		}
	}()
}

func PostComment() {
	commentEditor := document.GetElementByID("comment-editor").(*dom.HTMLTextAreaElement)

	value := commentEditor.Value
	if strings.TrimSpace(value) == "" {
		log.Println("cannot post empty comment")
		return
	}

	go func() {
		resp, err := http.PostForm(state.BaseURI+state.ReqPath+"/comment", url.Values{"csrf_token": {state.CSRFToken}, "value": {value}})
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Printf("got reply: %v\n%q\n", resp.Status, string(body))

		switch resp.StatusCode {
		case http.StatusOK:
			// Create comment.
			newComment := document.CreateElement("div").(*dom.HTMLDivElement)

			newItemMarker := document.GetElementByID("new-item-marker")
			newItemMarker.ParentNode().InsertBefore(newComment, newItemMarker)

			newComment.Underlying().Set("outerHTML", string(body))

			// Reset new-comment component.
			commentEditor.Value = ""
			SwitchWriteTab()
		}
	}()
}

var previewActive = false

func MarkdownPreview() {
	if previewActive {
		return
	}

	commentEditor := document.GetElementByID("comment-editor").(*dom.HTMLTextAreaElement)
	commentPreview := document.GetElementByID("comment-preview").(*dom.HTMLDivElement)

	in := commentEditor.Value
	if in == "" {
		in = "Nothing to preview."
	}
	commentPreview.SetInnerHTML(string(github_flavored_markdown.Markdown([]byte(in))))

	document.GetElementByID("write-tab-link").(dom.Element).Class().Remove("active")
	document.GetElementByID("preview-tab-link").(dom.Element).Class().Add("active")
	commentEditor.Style().SetProperty("display", "none", "")
	commentPreview.Style().SetProperty("display", "block", "")
	previewActive = true
}

func SwitchWriteTab() {
	if !previewActive {
		return
	}

	commentEditor := document.GetElementByID("comment-editor").(*dom.HTMLTextAreaElement)
	commentPreview := document.GetElementByID("comment-preview").(*dom.HTMLDivElement)

	document.GetElementByID("write-tab-link").(dom.Element).Class().Add("active")
	document.GetElementByID("preview-tab-link").(dom.Element).Class().Remove("active")
	commentEditor.Style().SetProperty("display", "block", "")
	commentPreview.Style().SetProperty("display", "none", "")
	previewActive = false
}
