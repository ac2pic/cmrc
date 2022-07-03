package repo

import "github.com/google/go-github/v45/github"

type Repository struct {
	Owner string
	Name string
	hashToManifestPaths map[string]map[string]string
	client * github.Client
}

func NewRepository(owner string, name string, client * github.Client) *Repository {
	return &Repository{Owner:owner, Name:name, client:client, hashToManifestPaths: make(map[string]map[string]string)}
}

type SerializableRepository struct {
	Owner string `json:"owner"`
	Name string `json:"repo"`
	Commits map[string]map[string]string `json:"commits"`
}

