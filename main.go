package main

import (
	"net/http"
	"net/url"
	"fmt"
	"strings"
	"github.com/google/go-github/v45/github"
	"context"
	"golang.org/x/oauth2"
	"os"
	repository "github.com/ac2pic/cmrc/internal/repo"
	"encoding/json"
	"bytes"
	"io"
	"time"
)

var ctx context.Context = context.TODO()

var sha1path_pair map[string]string

func fromGithubBlobToRaw(blobUrl string) string {
	u, err := url.Parse(blobUrl);
	if err != nil {
		return ""
	}
	if u.Hostname() != "github.com" {
		return ""
	}

	bp := strings.Split(u.Path, "/")
	if len(bp) < 6 {
		return ""
	}

	if bp[3] != "blob" {
		return ""
	}

	nbp := append(bp[0:3], bp[4:]...)

	u.Host = "raw.githubusercontent.com"
	u.Path = strings.Join(nbp, "/")
	return u.String()
}







type RepositoryTrackEntry struct {
	Owner string `json:"owner"`
	Name string `json:"repo"`
	Id string `json:"id"`
	ExploreRecursively bool `json:"recursive,omitempty"`
}

func findRepo(repoList []*repository.Repository, owner string, name string, id string) *repository.Repository {
	for _, repo := range repoList {
		if (repo.Owner == owner && repo.Name == name) || (id != "" && repo.Id == id) {
			return repo
		}
	}
	return nil
}

func checkForTrackingUpdates(repoList []*repository.Repository, client * github.Client) []*repository.Repository {
	var track []*RepositoryTrackEntry

	data, err := os.ReadFile("track.json")
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal(data, &track); err != nil {
		panic(err)
	}

	// check for new entries

	for _, val := range track {
		repo := findRepo(repoList, val.Owner, val.Name, val.Id)
		if repo == nil {
			new_repo := repository.NewRepository(val.Owner, val.Name, val.Id, val.ExploreRecursively, client)
			repoList = append(repoList, new_repo)
		} else {
			repo.Owner = val.Owner
			repo.Name = val.Name
			repo.Id = val.Id
			repo.ExploreRecursively = val.ExploreRecursively
		}
	}
	return repoList
}

func checkErrorPanicOrWait(rr *github.Response, err error) {
	if (rr.Rate.Remaining > 0) {
		panic(err)
	}

	waitTime := rr.Rate.Reset.Sub(time.Now())

	if (waitTime > time.Second * 0) {
		fmt.Println("Rate limit hit. Will be waiting", waitTime)
		time.Sleep(waitTime)
	}
}

func main() {

	token, err := os.ReadFile(".token")
	if err != nil {
		panic(err)
	}

	t := oauth2.Token{ AccessToken: strings.TrimSuffix(string(token), "\n"), }

	ts := oauth2.StaticTokenSource(&t)
	authTransport := &oauth2.Transport{
		Source: ts,
	}

	httpClient := &http.Client{Transport: authTransport}

	client := github.NewClient(httpClient)

	var repos []*repository.Repository

	fh,e := os.ReadFile("out.json")

	if e != nil {
		if !strings.Contains(e.Error(), "no such file or directory") {
			panic(e)
		} else {
			fh = []byte("[]")
		}
	}


	if err := json.Unmarshal(fh, &repos); err != nil {
		panic(err)
	}

	// Add missing client object
	for _, repo := range repos  {
		repo.AddClient(client)
	}


	// Check if something updated what to track
	fmt.Println("Checking for tracking updates...")

	repos = checkForTrackingUpdates(repos, client)
	update := false

	fmt.Println("Checking for repo updates...")
	for _, repo := range repos {
		var branches []*github.Branch = nil
		repo_path := repo.Owner + "/" + repo.Name
		fmt.Println("Checking...", repo_path)
		for branches == nil {
			b, resp, err := repo.GetBranches()
			if err != nil {
				checkErrorPanicOrWait(resp, err)
				continue
			}
			branches = b
		}


		repo_updated := false

		branchCommits := make(map[string]string)


		// Length needs to match
		// for them to be the same
		if len(branches) != len(repo.BranchCommits) {
			repo_updated = true
		}

		for _, branch := range branches {
			branchCommit := branch.GetCommit().GetSHA()
			branchName := branch.GetName()
			branchCommits[branchName] = branchCommit
			// all branchNames must have the same commit
			// to be the same
			if repo.BranchCommits[branchName] != branchCommit {
				repo_updated = true
			}

			if repo.SearchCommitsForManifests(branchCommit) {
				repo_updated = true
			}
		}

		repo.BranchCommits = branchCommits


		if repo_updated {
			update = true
			fmt.Println("Updated... " + repo_path)
		}
	}


	if update {
		fmt.Println("Updating local copy...")
		b, e := json.Marshal(repos)
		if e != nil {
			panic(e)
		}
	
		f, e := os.Create("out.json")
		if e != nil {
			panic(e)
		}
		defer f.Close()
	
		reader := bytes.NewReader(b)
	
		_, e = io.Copy(f, reader)
		if e != nil {
			panic(e)
		}
		os.Exit(0)
	}
	os.Exit(1)
}

