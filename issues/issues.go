package issues

import (
	"fmt"
	"html/template"
	"time"

	"golang.org/x/net/context"
)

type RepoSpec struct {
	Owner string
	Repo  string
}

type Service interface {
	ListByRepo(ctx context.Context, repo RepoSpec, opt interface{}) ([]Issue, error)

	Get(ctx context.Context, repo RepoSpec, id uint64) (Issue, error)

	ListComments(ctx context.Context, repo RepoSpec, id uint64, opt interface{}) ([]Comment, error)
	ListEvents(ctx context.Context, repo RepoSpec, id uint64, opt interface{}) ([]Event, error)

	CreateComment(ctx context.Context, repo RepoSpec, id uint64, comment Comment) (Comment, error)

	Create(ctx context.Context, repo RepoSpec, issue Issue) (Issue, error)

	Edit(ctx context.Context, repo RepoSpec, id uint64, ir IssueRequest) (Issue, error)

	// TODO: This doesn't belong here, does it?
	CurrentUser() User
}

// Issue represents an issue on a repository.
type Issue struct {
	ID    uint64
	State string
	Title string
	Comment
}

// Comment represents a comment left on an issue.
type Comment struct {
	User      User
	CreatedAt time.Time
	Body      string
}

// Event represents an event that occurred around an issue.
type Event struct {
	Actor     User
	CreatedAt time.Time
	Type      EventType
	Rename    *Rename
}

type EventType string

const (
	Reopened EventType = "reopened"
	Closed             = "closed"
	Renamed            = "renamed"
)

func (et EventType) Valid() bool {
	switch et {
	case Reopened, Closed, Renamed:
		return true
	default:
		return false
	}
}

type Rename struct {
	From string
	To   string
}

// User represents a user.
type User struct {
	Login     string
	AvatarURL template.URL
	HTMLURL   template.URL
}

// IssueRequest is a request to edit an issue.
type IssueRequest struct {
	State *string // TODO: Enum.
	Title *string
}

func (ir IssueRequest) Validate() error {
	if ir.State != nil {
		switch *ir.State {
		case "open", "closed":
		default:
			return fmt.Errorf("bad state")
		}
	}
	if ir.Title != nil {
		// TODO.
	}
	return nil
}
