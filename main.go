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
	"encoding/json"
	"io"
)

var ctx context.Context = context.TODO()

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

func findRepoManifests() {
}

func judgeTreeByStructure(tree *github.Tree) int32 {
	c := int32(0)
	tcw := map[string]int32{
		"assets": 100,
		"js": 25,
		"patches": 100,
		"node_modules": -100,
	}

	bcw := map[string]int32{
		"package.json": 50,
		"package-lock.json": -100,
	}

	for _, treeEntry := range tree.Entries {

		parts := strings.Split(treeEntry.GetPath(), "/")
		name := parts[len(parts) - 1]

		if treeEntry.GetType() == "tree" {
			if weight, ok := tcw[name]; ok {
				c += weight
			}
		} else if treeEntry.GetType() == "blob" {
			if weight, ok := bcw[name]; ok {
				c += weight
			}
		}
	}

	return c
}
func judgeManifestByKeys(manifest string) int32 {
	c := int32(0)
	props := make(map[string]interface{})

	if err := json.Unmarshal([]byte(manifest), &props); err != nil {
		panic("judgeManifestByKeys:" + err.Error())
	}

	kw := map[string]int32 {
		"main": 100,
		"version": 50,
		"preload": 200,
		"postload": 200,
		"prestart": 200,
		"displayName": 200,
		"ccmodHumanName": 10000,
		"devDependencies": -200,
		"scripts": -200,
	}

	for key := range props {
		if weight, ok := kw[key]; ok {
			c += weight
		}
	}
	return c
}

func findBranchManifest(client * github.Client, owner string, repo string, branch string) string {

	isRoot := true
	treeShas := []string{branch}
	treeShasIndex := 0

	candidates := []string{}
	confidence := []int32{}
	tn := map[string]string{}
	for ;treeShasIndex < len(treeShas);  {
		treeSha := treeShas[treeShasIndex]
		rt, _, err := client.Git.GetTree(ctx, owner, repo, treeSha, false)
	
		if err != nil {
			return ""
		}

		checkConfidence := false
		var canPak *github.TreeEntry = nil
		for _, treeEntry := range rt.Entries {
			tp := treeEntry.GetPath()
			fn := tp
			if !isRoot {
				fn = tn[treeSha] + "/" + tp
			}

			if treeEntry.GetType() == "tree" {
				if isRoot {
					tn[treeEntry.GetSHA()] = tp
					treeShas = append(treeShas, treeEntry.GetSHA())
				}
				continue
			}

			if tp ==  "ccmod.json" {
				return fn
			}

			if tp == "package.json" {
				canPak = treeEntry
				candidates = append(candidates, fn)
				checkConfidence = true
			}
		}

		if checkConfidence {
			fp := fmt.Sprintf("%s/%s", tn[treeSha], canPak.GetPath())

			baseUrl :="https://raw.githubusercontent.com/%s/%s/%s/%s" 
			fullUrl := fmt.Sprintf(baseUrl, owner, repo,branch, fp)

			r, err := http.Get(fullUrl)
			if err != nil {
				continue
			}
			o, err := io.ReadAll(r.Body)
			if err != nil {
				continue
			}
			mw := judgeManifestByKeys(string(o))
			sw := judgeTreeByStructure(rt)
			fmt.Println("Confidence of",fp ,"is", mw + sw)
			confidence = append(confidence, sw + mw)
		}

		isRoot = false
		treeShasIndex++
	}

	bestPath := ""
	bestConfidence := int32(-1000000)
	for i := 0; i < len(candidates); i++ {
		can := candidates[i]
		con := confidence[i]
		if bestConfidence < con {
			bestPath = can
			bestConfidence = con
		}
	}

	return bestPath
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

/**	branches,response, err := client.Repositories.ListBranches(ctx, "CCDirectlink", "AMCS-wdeps",nil)

	if err != nil {
		panic(err)
	}

	fmt.Println(response.Rate.Remaining)

	for _, branch := range branches {
		fmt.Println(branch.GetName())
	}**/

	fmt.Println(findBranchManifest(client, "CCDirectlink", "unified-steps", "master"))
}

