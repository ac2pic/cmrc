package repo

import "github.com/google/go-github/v45/github"

type Repository struct {
	Owner string
	Name string
	Client * github.Client
}

