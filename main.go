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
	Name string `json:"name"`
}

func findRepo(repoList []*repository.Repository, owner string, name string) *repository.Repository {
	for _, repo := range repoList {
		if repo.Owner == owner && repo.Name == name {
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
		repo := findRepo(repoList, val.Owner, val.Name)
		if repo == nil {
			repo := repository.NewRepository(val.Owner, val.Name, client)
			repoList = append(repoList, repo)
		} 
	}
	return repoList
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
		panic(e)
	}


	if err := json.Unmarshal(fh, &repos); err != nil {
		panic(err)
	}

	// Add missing client object
	for _, repo := range repos  {
		repo.AddClient(client)
	}


	for {


		// Check if something updated what to track
		fmt.Println("Checking for tracking updates...")

		repos = checkForTrackingUpdates(repos, client)
		update := false

		fmt.Println("Checking for repo updates...")
		for _, repo := range repos {
			updated := repo.UpdateAllBranchesManifestPaths()
			if updated {
				update = true
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
		} else {
			fmt.Println("Did not need to update local copy")
		}
		fmt.Println("Waiting 5 seconds...")
		time.Sleep(5 * time.Second)
	}

}

