package repo

import (
	"github.com/google/go-github/v45/github"
	"context"
	"encoding/json"
	"strings"
	"fmt"
	"errors"
	"time"
)



func (r * Repository) findManifest(tree * github.Tree, sha1 string, basePath string) (*github.TreeEntry, error) {
	var npm_manifest *github.TreeEntry = nil
	for _, treeEntry := range tree.Entries {
		tp := treeEntry.GetPath()

		if treeEntry.GetType() == "tree" {
			continue
		}
		if tp ==  "ccmod.json" {
			return treeEntry, nil
		}
		if tp == "package.json" {
			npm_manifest = treeEntry
		}
	}

	if npm_manifest != nil {
		fp := basePath + *npm_manifest.Path
		content, err := r.DownloadContent(sha1, fp)
		if err != nil {
			return nil, err
		}

		data := make(map[string]interface{})
		if err := json.Unmarshal([]byte(content), &data); err != nil {
			return nil, errors.New(fmt.Sprintf("findManifest: %s/%s is not a valid json file", sha1, fp))
		}
		confidence := judgeManifestByKeys(data)
		confidence += judgeTreeByStructure(tree)
		if confidence > 100 {
			return npm_manifest, nil
		}
	}
	return nil, nil
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



func (repo * Repository) createCcmodGitManifest(props map[string]interface{}, path string) (*GitManifest, error) {
	manifest := &GitManifest{Path: path}

	id, ok := props["id"]; 
	if !ok {
		return manifest, errors.New("createCcmodGitManifest: id not found")
	}

	strId, ok := id.(string);
	if !ok {
		return manifest, errors.New("createCcmodGitManifest: id is not a string")
	} 
	manifest.Id = strId
	
	version, ok := props["version"]; 
	if !ok {
		return manifest, errors.New("createCcmodGitManifest: version not found")
	}

	strVersion, ok := version.(string);
	if !ok {
		return manifest, errors.New("createCcmodGitManifest: version is not a string")
	} 

	manifest.Version = strVersion

	return manifest, nil
}

func (repo * Repository) createNodeGitManifest(props map[string]interface{}, path string) (*GitManifest, error) {
	manifest := &GitManifest{Path: path}
	id, ok := props["name"]; 
	if !ok {
		return manifest, errors.New("createNodeGitManifest: name not found")
	}

	strId, ok := id.(string);
	if !ok {
		return manifest, errors.New("createNodeGitManifest: name is not a string")
	} 
	manifest.Id = strId
	
	version, ok := props["version"]; 
	if !ok {
		return manifest, errors.New("createNodeGitManifest: version not found")
	}

	strVersion, ok := version.(string);
	if !ok {
		return manifest, errors.New("createNodeGitManifest: version is not a string")
	} 

	manifest.Version = strVersion

	return manifest, nil
}


func (repo * Repository) createGitManifest(sha1 string, fp string) (*GitManifest, error) {

	content, err := repo.DownloadContent(sha1, fp)

	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return &GitManifest{}, errors.New(fmt.Sprintf("findManifest: %s/%s is not a valid json file", sha1, fp))
	}

	if strings.HasSuffix(fp, "ccmod.json") {
		return repo.createCcmodGitManifest(data, fp)
	}
	return repo.createNodeGitManifest(data, fp)
}

func (repo * Repository) GetManifests(sha1 string) ([]*GitManifest, error) {
	// TODO: Change terrible variable names or break this function up smaller
	gitManifests := make([]*GitManifest, 0)
	treeShas := []string{sha1}
	tn := map[string]string{}
	tn[sha1] = ""


	for treeShasIndex := 0;treeShasIndex < len(treeShas);  {
		treeSha := treeShas[treeShasIndex]
		rt, rr, err := repo.client.Git.GetTree(context.TODO(), repo.Owner, repo.Name, treeSha, false)

		// Likely needs to be moved
		// As other code will use it
		if rr.Rate.Remaining == 0 {
			restTime := rr.Rate.Reset.Time.Sub(time.Now());
			time.Sleep(restTime)
		}

		if err != nil {
			return nil, err
		}

		if repo.ExploreRecursively || treeShasIndex == 0 {
			// add subtrees for exploration
			for _, subtree := range findSubtrees(rt) {
				subtreeSha := subtree.GetSHA()
				treeShas = append(treeShas, subtreeSha)
				tn[subtreeSha] = tn[treeSha] + subtree.GetPath() + "/"
			}
		}


		treeShasIndex++

		manifest, err := repo.findManifest(rt, sha1, tn[treeSha])

		if err != nil {
			fmt.Println(err)
			continue
		}

		if manifest == nil {
			continue
		}

		name := manifest.GetPath()

		fp := tn[treeSha] + name
		gitManifest, err := repo.createGitManifest(sha1, fp)

		if err != nil {
			panic("GetManifest: " + err.Error())
		}

		gitManifests = append(gitManifests, gitManifest)
	}

	return gitManifests, nil
}

func (r * Repository) GetBranches() ([]*github.Branch, *github.Response, error) {
	return r.client.Repositories.ListBranches(context.TODO(), r.Owner, r.Name, nil)
}

// For parent commits, check what files changed.
// If the manifest paths found did not receive any edits then assume
// the same version. Otherwise Get Manifest and update id and/or version.
// Commits with broken manifest get a null entry.

type respGitManifestUpdate struct {
	commit string
	parents []string
	manifests []*GitManifest
	updated []bool
	response *github.Response
	changed bool
}

func findManifestFilesInCommit(rc *github.RepositoryCommit, folder string) map[string]*github.CommitFile {
	files := make(map[string]bool)
	files[folder + "package.json"] = true
	files[folder + "ccmod.json"] = true
	manifests := make(map[string]*github.CommitFile)
	for _, file := range rc.Files {
		fn := file.GetFilename()
		if _, ok := files[fn]; ok {
			manifests[fn] = file
		}
	}
	return manifests
}

func (r * Repository) checkManifestChanges(commitSha string, manifests []*GitManifest, updated[]bool, out chan <-*respGitManifestUpdate) {

	rc, rr, err := r.client.Repositories.GetCommit(context.TODO(), r.Owner, r.Name, commitSha, nil)

	if err != nil {
		panic(err)
	}

	rg := &respGitManifestUpdate{commit: commitSha, parents:make([]string, 0), manifests: make([]*GitManifest, 0)}
	rg.response = rr

	if updated == nil {
		rg.manifests = manifests

		for i := len(manifests); i > 0; i-- {
			rg.updated = append(rg.updated, true)
		}

		manifests = []*GitManifest{}
	}

	for idx, manifest := range manifests {
		if manifest == nil {
			continue
		}

		folder := ""
		pieces := strings.Split(manifest.Path, "/")
		if len(pieces) > 1 {
			folder = strings.Join(pieces[0:len(pieces) - 1],"/") + "/"
		}

		cm := findManifestFilesInCommit(rc, folder)

		upd := false

		tfs := []string{folder + "ccmod.json", folder + "package.json"}


		for _, tf := range tfs {
			if val, ok := cm[tf]; ok {
				status := val.GetStatus()
				if status == "removed" {
					// search 
					continue
				}

				if status == "added" || status == "modified" {
					nm, err := r.createGitManifest(commitSha, tf)
					if err != nil {
						fmt.Println(err)
					} else {
						if nm.Id != manifest.Id || nm.Path != manifest.Path || nm.Version != manifest.Version {
							upd = true
							rg.manifests = append(rg.manifests, nm)
						}
					}
					break
				}
			} 
		}

		if upd {
			rg.updated = append(rg.updated, upd)
			continue
		}

		if !updated[idx] {
			rg.updated = append(rg.updated, upd)
			rg.manifests = append(rg.manifests, manifest)
			continue
		}


		var correct_manifest *GitManifest = nil

		for _, tf := range tfs {
			nm, err := r.createGitManifest(commitSha, tf)
			if err != nil {
				if strings.HasPrefix(err.Error(), "404") {
					continue
				} else if nm == nil {
					panic(err)
				} else if nm.Id == "" || nm.Version == "" {
					fmt.Println(err.Error())
					continue
				}
			}
			correct_manifest = nm
			break
		}

		rg.manifests = append(rg.manifests, correct_manifest)
		rg.updated = append(rg.updated, false)
	}

	for _, parent := range rc.Parents {
		rg.parents = append(rg.parents, parent.GetSHA())
	}

	out <- rg
}


func (r * Repository) SearchCommitsForManifests(branch *github.Branch) bool {

	rootCommit := branch.GetCommit().GetSHA()


	if _, ok := r.GitManifestsByCommit[rootCommit]; ok {
		return false
	}

	manifests, err := r.GetManifests(rootCommit)

	if err != nil {
		fmt.Println(err)
		return false
	}

	out := make(chan *respGitManifestUpdate, 0)

	go r.checkManifestChanges(rootCommit, manifests, nil, out)
	waitOn := 1
	updated := false
	for data := range out {
		waitOn -= 1
		_, ok := r.GitManifestsByCommit[data.commit]
		fmt.Println(data.commit, data.manifests)

		if !ok {
			updated = true
			r.GitManifestsByCommit[data.commit] = data.manifests
			rate := data.response.Rate
			prl := rate.Remaining
			for _, commitSha := range data.parents {
				prl -= 1
				if prl < 0 {
					fmt.Println("Rate limit hit")
					fmt.Println(time.Now())
					resetTime := rate.Reset.Sub(time.Now())
					fmt.Println("Will reset in", resetTime)
					time.Sleep(resetTime)
					prl = data.response.Rate.Limit - prl
				}
				waitOn += 1
				go r.checkManifestChanges(commitSha, data.manifests, data.updated, out)
			}
		}

		if waitOn == 0 {
			close(out)
		}
	}

	return updated

}

