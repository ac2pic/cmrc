package repo

import (
	"github.com/google/go-github/v45/github"
	"context"
	"encoding/json"
	"strings"
	"fmt"
)

func findManifest(tree * github.Tree) *github.TreeEntry {
	for _, treeEntry := range tree.Entries {
		tp := treeEntry.GetPath()

		if treeEntry.GetType() == "tree" {
			continue
		}
		if tp ==  "ccmod.json" || tp == "package.json" {
			return treeEntry
		}
	}
	return nil
}

func findSubtrees(tree * github.Tree) []*github.TreeEntry {
	subtrees := []*github.TreeEntry{}
	for _, treeEntry := range tree.Entries {
		if treeEntry.GetType() == "tree" {
			subtrees = append(subtrees, treeEntry)
		}
	}
	return subtrees
}

func judgeTreeByStructure(tree *github.Tree) int32 {
	confidentW := int32(100)
	veryUnconfidentW := int32(-100)

	c := int32(0)
	tcw := map[string]int32{
		"assets": confidentW,
		"js": confidentW,
		"patches": confidentW,
		"node_modules": veryUnconfidentW,
	}

	bcw := map[string]int32{
		"package.json": confidentW,
		"package-lock.json": veryUnconfidentW,
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
		fmt.Println(manifest)
		panic("judgeManifestByKeys:" + err.Error())
	}


	veryConfidentW := int32(10000)
	confidentW := int32(100)
	veryUnconfidentW := int32(-100)

	kw := map[string]int32 {
		"main": confidentW,
		"version": confidentW,
		"preload": veryConfidentW,
		"postload": veryConfidentW,
		"prestart": veryConfidentW,
		"displayName": veryConfidentW,
		"ccmodHumanName": veryConfidentW,
		"ccmodDependencies": veryConfidentW,
		"devDependencies":  veryUnconfidentW,
		"scripts": veryUnconfidentW,
	}

	for key := range props {
		if weight, ok := kw[key]; ok {
			c += weight
		}
	}
	return c
}

// func (repo * Repository) UpdateVersionToCommitHash
func (repo * Repository) UpdateManifestPaths(sha1 string) (bool, error) {


	if _, ok := repo.Commits[sha1]; ok {
		return false, nil
	}

	// TODO: Change terrible variable names or break this function up smaller
	subpathManifests := make([]string, 0)
	client := repo.client
	treeShas := []string{sha1}
	treeShasIndex := 0
	candidates := []string{}
	confidences := []int32{}
	tn := map[string]string{}
	tn[sha1] = ""


	for ;treeShasIndex < len(treeShas);  {
		treeSha := treeShas[treeShasIndex]
		rt, _, err := client.Git.GetTree(context.TODO(), repo.Owner, repo.Name, treeSha, false)
		
		if err != nil {
			return false, err
		}

		if treeShasIndex == 0 {
			// add subtrees for exploration
			for _, subtree := range findSubtrees(rt) {
				subtreeSha := subtree.GetSHA()
				treeShas = append(treeShas, subtreeSha)
				tn[subtreeSha] = subtree.GetPath() + "/"
			}
		}

		treeShasIndex++

		manifest := findManifest(rt)

		if manifest == nil {
			continue
		}

		name := manifest.GetPath()

		fp := tn[treeSha] + name
		if name == "ccmod.json"  {
			subpathManifests = append(subpathManifests, fp)
			continue
		}

		content, err := repo.DownloadContent(sha1, fp)
		if err != nil {
			return false, err

		}
		confidence := judgeManifestByKeys(string(content))
		confidence += judgeTreeByStructure(rt)
		confidences = append(confidences, confidence)
		candidates = append(candidates, fp)
	}

	for i := 0; i < len(candidates); i++ {
		can := candidates[i]
		con := confidences[i]
		if con > 100 {
			subpathManifests = append(subpathManifests, can)
		}
	}

	repo.Commits[sha1] = subpathManifests

	return true, nil
}


func (r * Repository) UpdateAllBranchesManifestPaths() bool {

	branches,_, err := r.client.Repositories.ListBranches(context.TODO(), r.Owner, r.Name,nil)

	if err != nil {
		return false
	}


	updated := false

	for _, branch := range branches {
		bsha := branch.GetCommit().GetSHA()

		u, err := r.UpdateManifestPaths(bsha)
		if err != nil {
			continue
		}
		if u {
			updated = true
		}
	}
	return updated
}

