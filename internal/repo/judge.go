package repo

import (
	"github.com/google/go-github/v45/github"
	"strings"
)

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

func judgeManifestByKeys(manifest map[string]interface{}) int32 {
	c := int32(0)

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

	for key := range manifest {
		if weight, ok := kw[key]; ok {
			c += weight
		}
	}
	return c
}

