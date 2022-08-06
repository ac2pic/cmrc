package repo

import (
	"net/http"
	"io"
	"fmt"
	"errors"
)

func (r * Repository) DownloadContent(branchOrSha1 string, path string) ([]byte, error) {

	url := "https://raw.githubusercontent.com/%s/%s/%s/%s"

	fullPath := fmt.Sprintf(url, r.Owner, r.Name, branchOrSha1, path)

	rsp, err := http.Get(fullPath)
	if err != nil {
		return []byte(""), err
	}

	if rsp.StatusCode != 200 {
		return []byte(""), errors.New(rsp.Status)
	}

	o, err := io.ReadAll(rsp.Body)
	if err != nil {
		return []byte(""), err
	}

	return o, nil
}
