// +build js

package main

import (
	"github.com/gopherjs/gopherjs/js"
	"github.com/shurcooL/github_flavored_markdown"
	"honnef.co/go/js/dom"
)

var document = dom.GetWindow().Document().(dom.HTMLDocument)

func main() {
	js.Global.Set("MarkdownPreview", MarkdownPreview)
	js.Global.Set("SwitchWriteTab", SwitchWriteTab)
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
