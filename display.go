package main

import "github.com/gopherjs/gopherpen/issues"

// event is a issues.Event wrapper with display augmentations.
type event struct {
	issues.Event
}

func (e event) Octicon() string {
	switch e.Event.Type {
	case issues.Reopened:
		return "octicon-primitive-dot"
	case issues.Closed:
		return "octicon-circle-slash"
	default:
		panic("unexpected event")
	}
}
