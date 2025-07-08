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
	require.Equal(t, "docker.io/foo", images[0].Repository)
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

// MKE3.8.7 forced us to refine the regex used. This test uses some real MKE output w/ --debug to confirm
func TestMKE387OutputTest(t *testing.T) {
	mke387output := `time="2025-07-03T01:00:12Z" level=debug msg="Verifying docker.sock"
time="2025-07-03T01:00:12Z" level=debug msg="Checking for compatible kernel version"
time="2025-07-03T01:00:12Z" level=debug msg="Kernel version 6.8.0-1030-aws is compatible"
time="2025-07-03T01:00:12Z" level=info msg="Skipping compatible engine version check for --force-engine-minimum"
time="2025-07-03T01:00:12Z" level=debug msg="Start finding bootstrap container"
time="2025-07-03T01:00:12Z" level=debug msg="Found 1 container(s) running the bootstrap image"
time="2025-07-03T01:00:12Z" level=debug msg="Container \"/priceless_edison\" running: /bin/ucp-tool images"
time="2025-07-03T01:00:12Z" level=info msg="Bootsrapper image org: msr.ci.mirantis.com/mirantiseng"
time="2025-07-03T01:00:12Z" level=info msg="Bootsrapper image version: 3.8.7"
msr.ci.mirantis.com/mirantiseng/ucp-agent:3.8.7
msr.ci.mirantis.com/mirantiseng/ucp-alertmanager:3.8.7
msr.ci.mirantis.com/mirantiseng/ucp-auth-store:3.8.7`
	images := AllFromString(mke387output)
	require.Equal(t, 3, len(images))
	require.Equal(t, "msr.ci.mirantis.com/mirantiseng/ucp-agent:3.8.7", images[0].String())
	require.Equal(t, "msr.ci.mirantis.com/mirantiseng/ucp-alertmanager:3.8.7", images[1].String())
	require.Equal(t, "msr.ci.mirantis.com/mirantiseng/ucp-auth-store:3.8.7", images[2].String())
}
