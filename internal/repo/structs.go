package repo

import "github.com/google/go-github/v45/github"
import "fmt"

type GitManifest struct {
	Path string `json:"path"`
	Id string `json:"id"`
	Version string `json:"version"`
}

func (g * GitManifest) String() string {
	return fmt.Sprintf("%s: %s %s", g.Path, g.Id, g.Version)
}

type Repository struct {
	Owner string `json:"owner"`
	Name string `json:"repo"`
	GitManifestsByCommit map[string][]*GitManifest `json:"manifests_by_commit"`
	client * github.Client
}

func NewRepository(owner string, name string, client * github.Client) *Repository {
	return &Repository{Owner:owner, Name:name, client:client, GitManifestsByCommit: make(map[string][]*GitManifest)}
}

func (r * Repository) AddClient(client * github.Client) {
	r.client = client
}

