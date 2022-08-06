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
	ExploreRecursively bool `json:"recursive,omitempty"`
	client * github.Client
}

func NewRepository(owner string, name string, recursive bool, client * github.Client) *Repository {
	return &Repository{Owner:owner, Name: name, ExploreRecursively: recursive, GitManifestsByCommit:make(map[string][]*GitManifest, 0), client:client} 
}
func (r * Repository) AddClient(client * github.Client) {
	r.client = client
}

