// Package github implements issues.Service using GitHub API client.
package github

import (
	"html/template"
	"time"

	"github.com/google/go-github/github"
	"github.com/gopherjs/gopherpen/issues"
	"golang.org/x/net/context"
)

func NewService(client *github.Client) issues.Service {
	if client == nil {
		client = github.NewClient(nil)
	}
	return service{
		cl: client,
	}
}

type service struct {
	cl *github.Client

	// TODO.
	issues.Service
}

func (s service) ListByRepo(_ context.Context, repo issues.RepoSpec, opt interface{}) ([]issues.Issue, error) {
	ghIssuesAndPRs, _, err := s.cl.Issues.ListByRepo(repo.Owner, repo.Repo, &github.IssueListByRepoOptions{State: "open"})
	if err != nil {
		return nil, err
	}

	var is []issues.Issue
	for _, issue := range ghIssuesAndPRs {
		// Filter out PRs.
		if issue.PullRequestLinks != nil {
			continue
		}

		is = append(is, issues.Issue{
			ID:    uint64(*issue.Number),
			State: *issue.State,
			Title: *issue.Title,
			Comment: issues.Comment{
				User: issues.User{
					Login:     *issue.User.Login,
					AvatarURL: template.URL(*issue.User.AvatarURL),
					HTMLURL:   template.URL(*issue.User.HTMLURL),
				},
				CreatedAt: *issue.CreatedAt,
			},
		})
	}

	return is, nil
}

func (s service) Get(_ context.Context, repo issues.RepoSpec, id uint64) (issues.Issue, error) {
	issue, _, err := s.cl.Issues.Get(repo.Owner, repo.Repo, int(id))
	if err != nil {
		return issues.Issue{}, err
	}

	return issues.Issue{
		ID:    uint64(*issue.Number),
		State: *issue.State,
		Title: *issue.Title,
		Comment: issues.Comment{
			User: issues.User{
				Login:     *issue.User.Login,
				AvatarURL: template.URL(*issue.User.AvatarURL),
				HTMLURL:   template.URL(*issue.User.HTMLURL),
			},
			CreatedAt: *issue.CreatedAt,
		},
	}, nil
}

func (s service) ListComments(_ context.Context, repo issues.RepoSpec, id uint64, opt interface{}) ([]issues.Comment, error) {
	var comments []issues.Comment

	issue, _, err := s.cl.Issues.Get(repo.Owner, repo.Repo, int(id))
	if err != nil {
		return comments, err
	}
	comments = append(comments, issues.Comment{
		User: issues.User{
			Login:     *issue.User.Login,
			AvatarURL: template.URL(*issue.User.AvatarURL),
			HTMLURL:   template.URL(*issue.User.HTMLURL),
		},
		CreatedAt: *issue.CreatedAt,
		Body:      *issue.Body,
	})

	ghComments, _, err := s.cl.Issues.ListComments(repo.Owner, repo.Repo, int(id), nil) // TODO: Pagination.
	if err != nil {
		return comments, err
	}
	for _, comment := range ghComments {
		comments = append(comments, issues.Comment{
			User: issues.User{
				Login:     *comment.User.Login,
				AvatarURL: template.URL(*comment.User.AvatarURL),
				HTMLURL:   template.URL(*comment.User.HTMLURL),
			},
			CreatedAt: *comment.CreatedAt,
			Body:      *comment.Body,
		})
	}

	return comments, nil
}

func (s service) ListEvents(_ context.Context, repo issues.RepoSpec, id uint64, opt interface{}) ([]issues.Event, error) {
	var events []issues.Event

	ghEvents, _, err := s.cl.Issues.ListIssueEvents(repo.Owner, repo.Repo, int(id), nil) // TODO: Pagination.
	if err != nil {
		return events, err
	}
	for _, event := range ghEvents {
		et := issues.EventType(*event.Event)
		if !et.Valid() {
			continue
		}
		e := issues.Event{
			Actor: issues.User{
				Login:     *event.Actor.Login,
				AvatarURL: template.URL(*event.Actor.AvatarURL),
				HTMLURL:   template.URL(*event.Actor.HTMLURL),
			},
			CreatedAt: *event.CreatedAt,
			Type:      et,
		}
		switch et {
		case issues.Renamed:
			e.Rename = &issues.Rename{
				From: *event.Rename.From,
				To:   *event.Rename.To,
			}
		}
		events = append(events, e)
	}

	return events, nil
}

func (service) Comment() issues.Comment {
	newInt := func(v int) *int {
		return &v
	}
	newString := func(v string) *string {
		return &v
	}
	newTime := func(v time.Time) *time.Time {
		return &v
	}

	comment := (github.IssueComment)(github.IssueComment{
		ID:        (*int)(newInt(143399786)),
		Body:      (*string)(newString("I've resolved this in 4387efb. Please re-open or leave a comment if there's still room for improvement here.")),
		User:      user(),
		CreatedAt: (*time.Time)(newTime(time.Unix(1443244474, 0).UTC())),
		UpdatedAt: (*time.Time)(newTime(time.Unix(1443244474, 0).UTC())),
		URL:       (*string)(newString("https://api.github.com/repos/shurcooL/vfsgen/issues/comments/143399786")),
		HTMLURL:   (*string)(newString("https://github.com/shurcooL/vfsgen/issues/8#issuecomment-143399786")),
		IssueURL:  (*string)(newString("https://api.github.com/repos/shurcooL/vfsgen/issues/8")),
	})

	return issues.Comment{
		User: issues.User{
			Login:     *comment.User.Login,
			AvatarURL: template.URL(*comment.User.AvatarURL),
			HTMLURL:   template.URL(*comment.User.HTMLURL),
		},
		CreatedAt: *comment.CreatedAt,
		Body:      *comment.Body,
	}
}

func (service) CurrentUser() issues.User {
	user := user()

	return issues.User{
		Login:     *user.Login,
		AvatarURL: template.URL(*user.AvatarURL),
		HTMLURL:   template.URL(*user.HTMLURL),
	}
}

func user() *github.User {
	newInt := func(v int) *int {
		return &v
	}
	newBool := func(v bool) *bool {
		return &v
	}
	newString := func(v string) *string {
		return &v
	}

	return (*github.User)(&github.User{
		Login:             (*string)(newString("shurcooL")),
		ID:                (*int)(newInt(1924134)),
		AvatarURL:         (*string)(newString("https://avatars.githubusercontent.com/u/1924134?v=3")),
		HTMLURL:           (*string)(newString("https://github.com/shurcooL")),
		GravatarID:        (*string)(newString("")),
		Name:              (*string)(nil),
		Company:           (*string)(nil),
		Blog:              (*string)(nil),
		Location:          (*string)(nil),
		Email:             (*string)(nil),
		Hireable:          (*bool)(nil),
		Bio:               (*string)(nil),
		PublicRepos:       (*int)(nil),
		PublicGists:       (*int)(nil),
		Followers:         (*int)(nil),
		Following:         (*int)(nil),
		CreatedAt:         (*github.Timestamp)(nil),
		UpdatedAt:         (*github.Timestamp)(nil),
		Type:              (*string)(newString("User")),
		SiteAdmin:         (*bool)(newBool(false)),
		TotalPrivateRepos: (*int)(nil),
		OwnedPrivateRepos: (*int)(nil),
		PrivateGists:      (*int)(nil),
		DiskUsage:         (*int)(nil),
		Collaborators:     (*int)(nil),
		Plan:              (*github.Plan)(nil),
		URL:               (*string)(newString("https://api.github.com/users/shurcooL")),
		EventsURL:         (*string)(newString("https://api.github.com/users/shurcooL/events{/privacy}")),
		FollowingURL:      (*string)(newString("https://api.github.com/users/shurcooL/following{/other_user}")),
		FollowersURL:      (*string)(newString("https://api.github.com/users/shurcooL/followers")),
		GistsURL:          (*string)(newString("https://api.github.com/users/shurcooL/gists{/gist_id}")),
		OrganizationsURL:  (*string)(newString("https://api.github.com/users/shurcooL/orgs")),
		ReceivedEventsURL: (*string)(newString("https://api.github.com/users/shurcooL/received_events")),
		ReposURL:          (*string)(newString("https://api.github.com/users/shurcooL/repos")),
		StarredURL:        (*string)(newString("https://api.github.com/users/shurcooL/starred{/owner}{/repo}")),
		SubscriptionsURL:  (*string)(newString("https://api.github.com/users/shurcooL/subscriptions")),
		TextMatches:       ([]github.TextMatch)(nil),
		Permissions:       (*map[string]bool)(nil),
	})
}
