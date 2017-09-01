package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/shurcooL/issues"
	"github.com/shurcooL/reactions"
)

// IssuesReactions implements reactions.Service on top of issues.Service,
// specifically for use by issuesapp.
//
// The format of ID is "{{.issueID}}/{{.commentID}}".
type IssuesReactions struct {
	Issues issues.Service
}

// Toggle toggles an issue.
// id is "{{.issueID}}/{{.commentID}}".
func (ir IssuesReactions) Toggle(ctx context.Context, uri string, id string, tr reactions.ToggleRequest) ([]reactions.Reaction, error) {
	var issueID, commentID uint64
	_, err := fmt.Sscanf(id, "%d/%d", &issueID, &commentID)
	if err != nil {
		return nil, err
	}
	comment, err := ir.Issues.EditComment(ctx, issues.RepoSpec{URI: uri}, issueID, issues.CommentRequest{
		ID:       commentID,
		Reaction: &tr.Reaction,
	})
	if err != nil {
		return nil, err
	}
	return comment.Reactions, nil
}

func (ir IssuesReactions) Get(_ context.Context, uri string, id string) ([]reactions.Reaction, error) {
	return nil, errors.New("IssuesReactions.Get: not implemented")
}

func (ir IssuesReactions) List(_ context.Context, uri string) (map[string][]reactions.Reaction, error) {
	return nil, errors.New("IssuesReactions.List: not implemented")
}
