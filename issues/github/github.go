// Package github implements issues.Service using GitHub API client.
package github

import (
	"html/template"

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

func (s service) CreateComment(_ context.Context, repo issues.RepoSpec, id uint64, c issues.Comment) (issues.Comment, error) {
	comment, _, err := s.cl.Issues.CreateComment(repo.Owner, repo.Repo, int(id), &github.IssueComment{
		Body: &c.Body,
	})
	if err != nil {
		return issues.Comment{}, err
	}

	return issues.Comment{
		User: issues.User{
			Login:     *comment.User.Login,
			AvatarURL: template.URL(*comment.User.AvatarURL),
			HTMLURL:   template.URL(*comment.User.HTMLURL),
		},
		CreatedAt: *comment.CreatedAt,
		Body:      *comment.Body,
	}, nil
}

func (s service) Create(_ context.Context, repo issues.RepoSpec, i issues.Issue) (issues.Issue, error) {
	issue, _, err := s.cl.Issues.Create(repo.Owner, repo.Repo, &github.IssueRequest{
		Title: &i.Title,
		Body:  &i.Body,
	})
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

func (s service) Edit(_ context.Context, repo issues.RepoSpec, id uint64, ir issues.IssueRequest) (issues.Issue, error) {
	if err := ir.Validate(); err != nil {
		return issues.Issue{}, err
	}

	issue, _, err := s.cl.Issues.Edit(repo.Owner, repo.Repo, int(id), &github.IssueRequest{
		State: ir.State,
		Title: ir.Title,
	})
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

func (service) CurrentUser() issues.User {
	return issues.User{
		Login:     "shurcooL",
		AvatarURL: "https://avatars.githubusercontent.com/u/1924134?v=3",
		HTMLURL:   "https://github.com/shurcooL",
	}
}