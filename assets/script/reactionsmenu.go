// +build js

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gopherjs/gopherjs/js"
	"github.com/shurcooL/reactions"
	"honnef.co/go/js/dom"
)

var Reactions ReactionsMenu

func (rm *ReactionsMenu) Show(this dom.HTMLElement, event dom.Event, commentID uint64) {
	rm.commentID = commentID

	rm.filter.Value = ""
	rm.filter.Underlying().Call("dispatchEvent", js.Global.Get("CustomEvent").New("input")) // Trigger "input" event listeners.
	updateSelected(0)

	rm.menu.Style().SetProperty("display", "initial", "")

	// Reactions.menu aims to have 270px client width. Due to optional scrollbars
	// taking up some of that space, we may need to compensate and increase width.
	if scrollbarWidth := Reactions.results.OffsetWidth() - Reactions.results.Get("clientWidth").Float(); scrollbarWidth > 0 {
		Reactions.menu.Style().SetProperty("width", fmt.Sprintf("%fpx", 270+scrollbarWidth+1), "")
	}

	rm.results.Set("scrollTop", 0)
	top := float64(dom.GetWindow().ScrollY()) + this.GetBoundingClientRect().Top - rm.menu.GetBoundingClientRect().Height - 10
	if minTop := float64(dom.GetWindow().ScrollY()) + 12; top < minTop {
		top = minTop
	}
	rm.menu.Style().SetProperty("top", fmt.Sprintf("%vpx", top), "")
	left := float64(dom.GetWindow().ScrollX()) + this.GetBoundingClientRect().Left
	if maxLeft := float64(dom.GetWindow().InnerWidth()+dom.GetWindow().ScrollX()) - rm.menu.GetBoundingClientRect().Width - 12; left > maxLeft {
		left = maxLeft
	}
	if minLeft := float64(dom.GetWindow().ScrollX()) + 12; left < minLeft {
		left = minLeft
	}
	rm.menu.Style().SetProperty("left", fmt.Sprintf("%vpx", left), "")
	if rm.authenticatedUser {
		rm.filter.Focus()
	}

	event.PreventDefault()
}

func (rm *ReactionsMenu) hide() {
	rm.menu.Style().SetProperty("display", "none", "")
}

type ReactionsMenu struct {
	commentID uint64 // commentID from last Show.

	menu    *dom.HTMLDivElement
	filter  *dom.HTMLInputElement
	results *dom.HTMLDivElement

	authenticatedUser bool
}

func setupReactionsMenu() {
	Reactions.authenticatedUser = state.CurrentUser.ID != 0

	Reactions.menu = document.CreateElement("div").(*dom.HTMLDivElement)
	Reactions.menu.SetID("rm-reactions-menu")

	container := document.CreateElement("div").(*dom.HTMLDivElement)
	container.SetClass("rm-reactions-menu-container")
	Reactions.menu.AppendChild(container)

	// Disable for unauthenticated user.
	if !Reactions.authenticatedUser {
		disabled := document.CreateElement("div").(*dom.HTMLDivElement)
		disabled.SetClass("rm-reactions-menu-disabled")
		signIn := document.CreateElement("div").(*dom.HTMLDivElement)
		signIn.SetClass("rm-reactions-menu-signin")
		signIn.SetInnerHTML(fmt.Sprintf(`<form method="post" action="/login/github" style="display: inline-block; margin-bottom: 0;"><input class="btn" type="submit" value="Sign in via GitHub"><input type="hidden" name="return" value="%s"></form> to react.`, state.BaseURI+state.ReqPath))
		disabled.AppendChild(signIn)
		container.AppendChild(disabled)
	}

	Reactions.filter = document.CreateElement("input").(*dom.HTMLInputElement)
	Reactions.filter.SetClass("rm-reactions-filter")
	Reactions.filter.Placeholder = "Search"
	Reactions.menu.AddEventListener("click", false, func(event dom.Event) {
		if Reactions.authenticatedUser {
			Reactions.filter.Focus()
		}
	})
	container.AppendChild(Reactions.filter)
	Reactions.results = document.CreateElement("div").(*dom.HTMLDivElement)
	Reactions.results.SetClass("rm-reactions-results")
	Reactions.results.AddEventListener("click", false, func(event dom.Event) {
		me := event.(*dom.MouseEvent)
		x := (me.ClientX - int(Reactions.results.GetBoundingClientRect().Left) + Reactions.results.Underlying().Get("scrollLeft").Int()) / 30
		if x >= 9 {
			return // Out of bounds to the right, likely because of scrollbar.
		}
		y := (me.ClientY - int(Reactions.results.GetBoundingClientRect().Top) + Reactions.results.Underlying().Get("scrollTop").Int()) / 30
		i := y*9 + x
		if i < 0 || i >= len(filtered) {
			return
		}
		emojiID := filtered[i]
		go func() {
			err := postReaction(strings.Trim(emojiID, ":"), Reactions.commentID)
			if err != nil {
				log.Println(err)
				return
			}
		}()
		Reactions.hide()
	})
	container.AppendChild(Reactions.results)
	preview := document.CreateElement("div").(*dom.HTMLDivElement)
	preview.SetClass("rm-reactions-preview")
	preview.SetInnerHTML(`<span id="rm-reactions-preview-emoji"><span class="rm-emoji rm-large"></span></span><span id="rm-reactions-preview-label"></span>`)
	container.AppendChild(preview)

	updateFilteredResults(Reactions.filter, Reactions.results)
	Reactions.filter.AddEventListener("input", false, func(dom.Event) {
		updateFilteredResults(Reactions.filter, Reactions.results)
	})

	Reactions.results.AddEventListener("mousemove", false, func(event dom.Event) {
		me := event.(*dom.MouseEvent)
		x := (me.ClientX - int(Reactions.results.GetBoundingClientRect().Left) + Reactions.results.Underlying().Get("scrollLeft").Int()) / 30
		if x >= 9 {
			return // Out of bounds to the right, likely because of scrollbar.
		}
		y := (me.ClientY - int(Reactions.results.GetBoundingClientRect().Top) + Reactions.results.Underlying().Get("scrollTop").Int()) / 30
		i := y*9 + x
		updateSelected(i)
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

	document.AddEventListener("click", false, func(event dom.Event) {
		if event.DefaultPrevented() {
			return
		}

		if !Reactions.menu.Contains(event.Target()) {
			Reactions.hide()
		}
	})
}

var filtered []string

func updateFilteredResults(filter *dom.HTMLInputElement, results dom.Element) {
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

// updateSelected reaction to filtered[index].
func updateSelected(index int) {
	if index < 0 || index >= len(filtered) {
		return
	}
	emojiID := filtered[index]

	label := document.GetElementByID("rm-reactions-preview-label").(*dom.HTMLSpanElement)
	label.SetTextContent(strings.Trim(emojiID, ":"))
	emoji := document.GetElementByID("rm-reactions-preview-emoji").(*dom.HTMLSpanElement)
	emoji.FirstChild().(dom.HTMLElement).Style().SetProperty("background-position", reactions.Position(emojiID), "")
}

func (rm *ReactionsMenu) ToggleReaction(this dom.HTMLElement, event dom.Event, emojiID string) {
	container := getAncestorByClassName(this, "comment-edit-container")
	// HACK: Currently the child nodes are [text, div, text, div, text], but that isn't reliable.
	editView := container.ChildNodes()[3].(dom.HTMLElement)
	commentEditor := editView.QuerySelector(".comment-editor").(*dom.HTMLTextAreaElement)
	commentID, _ := strconv.ParseUint(commentEditor.GetAttribute("data-id"), 10, 64)

	if !rm.authenticatedUser {
		rm.Show(this, event, commentID)
		return
	}

	go func() {
		err := postReaction(emojiID, commentID)
		if err != nil {
			log.Println(err)
			return
		}
	}()
}

func postReaction(emojiID string, commentID uint64) error {
	resp, err := http.PostForm(state.BaseURI+state.ReqPath+fmt.Sprintf("/comment/%v/react", commentID), url.Values{"reaction": {emojiID}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		reactionsContainer := document.GetElementByID(fmt.Sprintf("comment-%v-reactions-container", commentID)).(dom.HTMLElement)
		reactionsContainer.SetInnerHTML(string(body))
		return nil
	default:
		return fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, string(body))
	}
}
