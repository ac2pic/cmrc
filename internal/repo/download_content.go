package repo

import (
	"net/http"
	"io"
	"fmt"
)

func (r * Repository) DownloadContent(branchOrSha1 string, path string) ([]byte, error) {

	url := "https://raw.githubusercontent.com/%s/%s/%s/%s"

	fullPath := fmt.Sprintf(url, r.Owner, r.Name, branchOrSha1, path)

	rsp, err := http.Get(fullPath)
	if err != nil {
		return []byte(""), err
	}

	o, err := io.ReadAll(rsp.Body)
	if err != nil {
		return []byte(""), err
	}

	return o, nil
}
