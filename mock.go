package issuesapp

import (
	"log"
	"net/http"
	"path"
	"time"

	"github.com/shurcooL/issues"
	"github.com/shurcooL/issuesapp/common"
	"github.com/shurcooL/users"
)

func (h *handler) mockHandler(w http.ResponseWriter, req *http.Request) {
	if err := h.loadTemplates(common.State{}); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := path.Base(req.URL.Path)
	baseState, err := h.baseState(req)
	if err != nil {
		log.Println("baseState:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	mock := mockState{
		state: state{
			BaseState: baseState,
		},
	}
	err = t.ExecuteTemplate(w, tmpl+".tmpl", &mock)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type mockState struct {
	state
}

func (s mockState) Issue() issues.Issue {
	return issues.Issue{
		ID:      123,
		State:   issues.OpenState,
		Title:   "Mock issue.",
		Comment: s.Comment(),
		Replies: 0,
	}
}

func (s mockState) Comment() issues.Comment {
	return issues.Comment{
		User:      mockUser,
		CreatedAt: time.Unix(1443244474, 0).UTC(),
		Body:      "I've resolved this in [`4387efb`](https://www.example.com). Please re-open or leave a comment if there's still room for improvement here.",
	}
}

func (s mockState) Event() issues.Event {
	return issues.Event{
		Actor:     mockUser,
		CreatedAt: time.Now(),
		Type:      issues.Closed,
	}
}

var mockUser = users.User{
	Login:     "shurcooL",
	AvatarURL: "https://avatars.githubusercontent.com/u/1924134?v=3&s=96",
	HTMLURL:   "https://github.com/shurcooL",
}
