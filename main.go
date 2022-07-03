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






func findRepoManifests(client * github.Client, repo * repository.Repository) map[string]string {
	branches,_, err := client.Repositories.ListBranches(ctx, repo.Owner, repo.Name,nil)

	if err != nil {
		panic(err)
	}

	bmp := make(map[string]string)

	for _, branch := range branches {
		bsha := branch.GetCommit().GetSHA()

		mp := repo.GetManifestPath(bsha)
		bmp[bsha] = mp
	}
	return bmp

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

	repo := &repository.Repository{Owner: "ac2pic",Name: "emilie", Client:client}

	fmt.Println(findRepoManifests(client, repo))
}

