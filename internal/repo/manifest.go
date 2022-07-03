package repo

import (
	"github.com/google/go-github/v45/github"
	"context"
	"fmt"
	"encoding/json"
	"strings"
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
		"ccmodDependencies": 10000,
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

func (repo * Repository) GetManifestPath(branchOrSha1 string) string {

	client := repo.Client

	treeShas := []string{branchOrSha1}
	treeShasIndex := 0

	candidates := []string{}
	confidences := []int32{}
	tn := map[string]string{}
	tn[branchOrSha1] = ""
	for ;treeShasIndex < len(treeShas);  {
		treeSha := treeShas[treeShasIndex]
		rt, _, err := client.Git.GetTree(context.TODO(), repo.Owner, repo.Name, treeSha, false)
		
		if err != nil {
			fmt.Println(err)
			return ""
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
			return fp
		}

		content, err := repo.DownloadContent(treeSha, fp)
		if err != nil {
			continue
		}
		confidence := judgeManifestByKeys(string(content))
		confidence += judgeTreeByStructure(rt)
		confidences = append(confidences, confidence)
		candidates = append(candidates, fp)
	}

	bestPath := ""
	bestConfidence := int32(-1000000)
	for i := 0; i < len(candidates); i++ {
		can := candidates[i]
		con := confidences[i]
		if bestConfidence < con {
			bestPath = can
			bestConfidence = con
		}
	}

	return bestPath
}


