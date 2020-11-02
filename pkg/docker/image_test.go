package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewImage(t *testing.T) {
	image := NewImage("xyz/foofoo:1.2.3-latest")
	require.Equal(t, "xyz", image.Repository)
	require.Equal(t, "foofoo", image.Name)
	require.Equal(t, "1.2.3-latest", image.Tag)
	require.Equal(t, "xyz/foofoo:1.2.3-latest", image.String())

	image = NewImage("docker.io/xyz/foofoo:1.2.3-latest")
	require.Equal(t, "docker.io/xyz", image.Repository)
	require.Equal(t, "foofoo", image.Name)
	require.Equal(t, "1.2.3-latest", image.Tag)
	require.Equal(t, "docker.io/xyz/foofoo:1.2.3-latest", image.String())
}

func TestAllFromString(t *testing.T) {
	images := AllFromString(`docker.io/foo/bar:1.2.3
docker.io/foo/bar2:1.2.3`)
	require.Equal(t, 2, len(images))
	require.Equal(t, "docker.io/foo/bar:1.2.3", images[0].String())
	require.Equal(t, "docker.io/foo/bar2:1.2.3", images[1].String())
}

func TestAllToRepository(t *testing.T) {
	images := AllFromString(`docker.io/foo/bar:1.2.3
docker.io/foo/bar2:1.2.3`)
	moved := AllToRepository(images, "custom.example.com/repo")
	require.Equal(t, 2, len(moved))
	require.Equal(t, "custom.example.com/repo/bar:1.2.3", moved[0].String())
	require.Equal(t, "custom.example.com/repo/bar2:1.2.3", moved[1].String())
}
