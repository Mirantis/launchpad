package phase

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPullImagesCustomRepo(t *testing.T) {
	phase := PullImages{}
	require.Equal(t, "xyz/foofoo:1.2.3-latest", phase.ImageFromCustomRepo("docker/foofoo:1.2.3-latest", "xyz"))
	require.Equal(t, "xyz/foofoo:1.2.3-latest", phase.ImageFromCustomRepo("foo.example.com/docker/foofoo:1.2.3-latest", "xyz"))
}
