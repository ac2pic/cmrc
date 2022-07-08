package repo

import "github.com/google/go-github/v45/github"

type Repository struct {
	Owner string `json:"owner"`
	Name string `json:"repo"`
	Commits map[string][]string `json:"commits"`
	client * github.Client
}

func NewRepository(owner string, name string, client * github.Client) *Repository {
	return &Repository{Owner:owner, Name:name, client:client, Commits: make(map[string][]string)}
}

func (r * Repository) AddClient(client * github.Client) {
	r.client = client
}

