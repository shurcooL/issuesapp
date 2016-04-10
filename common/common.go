package common

import "github.com/shurcooL/issues"

type State struct {
	BaseURI     string
	ReqPath     string
	CurrentUser *issues.User
}
