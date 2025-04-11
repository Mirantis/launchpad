package docker

import (
	"strings"
)

// CompareRepos to repo registries to see if they are the same.
func CompareRepos(one, two string) bool {
	if one == "" {
		one = "docker.io"
	}
	if two == "" {
		two = "docker.io"
	}
	return strings.Split(strings.ToLower(one), "/")[0] == strings.Split(strings.ToLower(two), "/")[0]
}
