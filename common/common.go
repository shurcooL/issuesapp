package common

import "github.com/shurcooL/users"

type State struct {
	BaseURI     string
	ReqPath     string
	CurrentUser users.User
}
