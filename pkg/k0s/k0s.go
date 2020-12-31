package k0s

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// DownloadK0s gets k0s binaries from github
func DownloadK0s(version, arch string) (string, error) {
	switch arch {
	case "x86_64":
		arch = "amd64"
	case "aarch64":
		arch = "arm64"
	}

	url := fmt.Sprintf("https://github.com/k0sproject/k0s/releases/download/v%s/k0s-v%s-%s", version, version, arch)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if err == nil {
			return "", fmt.Errorf("Failed to get k0s binary (%d)", resp.StatusCode)
		}
		return "", err
	}

	out, err := ioutil.TempFile("", "k0s")
	if err != nil {
		return "", err
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	fmt.Println(out.Close())

	return out.Name(), nil
}
