// +build js

package main

import (
	"fmt"
	"strings"

	"github.com/gopherjs/gopherjs/js"
	"github.com/shurcooL/reactions"
	"honnef.co/go/js/dom"
)

func ShowReactionMenu(this dom.HTMLElement, commentID uint64) {
	Reactions.filter.Value = ""
	Reactions.filter.Underlying().Call("dispatchEvent", js.Global.Get("CustomEvent").New("input")) // Trigger "input" event listeners.

	Reactions.menu.Style().SetProperty("display", "initial", "")

	top := this.GetBoundingClientRect().Top - Reactions.menu.GetBoundingClientRect().Height - 8
	if top < 10 {
		top = 10
	}
	Reactions.menu.Style().SetProperty("top", fmt.Sprint(top), "")
	Reactions.filter.Focus()
}

var Reactions struct {
	menu   *dom.HTMLDivElement
	filter *dom.HTMLInputElement
}

func setupReactionsMenu() {
	Reactions.menu = document.CreateElement("div").(*dom.HTMLDivElement)
	Reactions.menu.SetID("rm-reactions-menu")

	container := document.CreateElement("div")
	Reactions.menu.AppendChild(container)

	Reactions.filter = document.CreateElement("input").(*dom.HTMLInputElement)
	Reactions.filter.SetClass("rm-reactions-filter")
	Reactions.filter.Placeholder = "Search"
	container.AppendChild(Reactions.filter)
	results := document.CreateElement("div").(*dom.HTMLDivElement)
	results.SetClass("rm-reactions-results")
	container.AppendChild(results)
	preview := document.CreateElement("div").(*dom.HTMLDivElement)
	container.AppendChild(preview)
	preview.SetOuterHTML(`<div class="rm-reactions-preview"><span id="rm-reactions-preview-emoji"></span><span id="rm-reactions-preview-label"></span></div>`)

	update(Reactions.filter, results)
	Reactions.filter.AddEventListener("input", false, func(dom.Event) {
		update(Reactions.filter, results)
	})

	results.AddEventListener("mousemove", false, func(event dom.Event) {
		me := event.(*dom.MouseEvent)
		x := (me.ClientX - int(results.GetBoundingClientRect().Left) + results.Underlying().Get("scrollLeft").Int()) / 30
		y := (me.ClientY - int(results.GetBoundingClientRect().Top) + results.Underlying().Get("scrollTop").Int()) / 30
		i := y*9 + x
		if i < 0 || i >= len(filtered) {
			return
		}
		emojiID := filtered[i]

		label := document.GetElementByID("rm-reactions-preview-label").(*dom.HTMLSpanElement)
		label.SetTextContent(strings.Trim(emojiID, ":"))
		emoji := document.GetElementByID("rm-reactions-preview-emoji").(*dom.HTMLSpanElement)
		emoji.SetInnerHTML(`<span class="rm-emoji rm-large" style="background-position: ` + reactions.Position(emojiID) + `;"></span></div>`)
	})

	document.AddEventListener("keydown", false, func(event dom.Event) {
		if event.DefaultPrevented() {
			return
		}
		// Ignore when some element other than body has focus (it means the user is typing elsewhere).
		/*if !event.Target().IsEqualNode(document.Body()) {
			return
		}*/

		switch ke := event.(*dom.KeyboardEvent); {
		// Escape.
		case ke.KeyCode == 27 && !ke.Repeat && !ke.CtrlKey && !ke.AltKey && !ke.MetaKey && !ke.ShiftKey:
			if Reactions.menu.Style().GetPropertyValue("display") == "none" {
				return
			}

			Reactions.menu.Style().SetProperty("display", "none", "")

			ke.PreventDefault()
		}
	})

	document.Body().AppendChild(Reactions.menu)
}

var filtered []string

func update(filter *dom.HTMLInputElement, results dom.Element) {
	lower := strings.ToLower(strings.TrimSpace(filter.Value))
	results.SetInnerHTML("")
	filtered = nil
	for _, emojiID := range reactions.Sorted {
		if lower != "" && !strings.Contains(emojiID, lower) {
			continue
		}
		element := document.CreateElement("div")
		results.AppendChild(element)
		element.SetOuterHTML(`<div class="rm-reaction"><span class="rm-emoji" style="background-position: ` + reactions.Position(emojiID) + `;"></span></div>`)
		filtered = append(filtered, emojiID)
	}
}
