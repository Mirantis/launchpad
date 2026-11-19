package docker_test

import (
	"testing"

	"github.com/Mirantis/launchpad/pkg/docker"
	"github.com/stretchr/testify/require"
)

func TestNewImage(t *testing.T) {
	t.Run("valid references", func(t *testing.T) {
		cases := []struct {
			input string
			repo  string
			name  string
			tag   string
			want  string
		}{
			{input: "xyz/foofoo:1.2.3-latest", repo: "xyz", name: "foofoo", tag: "1.2.3-latest", want: "xyz/foofoo:1.2.3-latest"},
			{input: "docker.io/xyz/foofoo:1.2.3-latest", repo: "docker.io/xyz", name: "foofoo", tag: "1.2.3-latest", want: "docker.io/xyz/foofoo:1.2.3-latest"},
			{input: "docker.io:8888/foo2/bar3:PRODENG", repo: "docker.io:8888/foo2", name: "bar3", tag: "PRODENG", want: "docker.io:8888/foo2/bar3:PRODENG"},
			{input: "docker.io/mirantis/ucp:3.7.0", repo: "docker.io/mirantis", name: "ucp", tag: "3.7.0", want: "docker.io/mirantis/ucp:3.7.0"},
			// An omitted tag is "latest", as it is for docker and for
			// ImageRepoAndTag, which does this half of the parsing.
			{input: "docker.io/mirantis/ucp", repo: "docker.io/mirantis", name: "ucp", tag: "latest", want: "docker.io/mirantis/ucp:latest"},
			// The colon here is the registry port, not a tag separator.
			{input: "registry.example.com:5000/mirantis/ucp", repo: "registry.example.com:5000/mirantis", name: "ucp", tag: "latest", want: "registry.example.com:5000/mirantis/ucp:latest"},
		}

		for _, tc := range cases {
			t.Run(tc.input, func(t *testing.T) {
				image, err := docker.NewImage(tc.input)
				require.NoError(t, err)
				require.Equal(t, tc.repo, image.Repository)
				require.Equal(t, tc.name, image.Name)
				require.Equal(t, tc.tag, image.Tag)
				require.Equal(t, tc.want, image.String())
			})
		}
	})

	// A malformed reference must produce an error, not a panic. Every input
	// here panicked before: the slicing in NewImage assumed a slash and a
	// colon in that order and checked neither.
	t.Run("malformed references", func(t *testing.T) {
		cases := []struct {
			desc  string
			input string
		}{
			{desc: "empty", input: ""},
			{desc: "no slash and no colon", input: "foo"},
			{desc: "no slash, with colon", input: "foo:1.2.3"},
			{desc: "colon only", input: ":"},
			{desc: "slash only", input: "/"},
			{desc: "slash in tag", input: "docker.io/mirantis/ucp:1.0/x"},
			{desc: "empty name", input: "docker.io/mirantis/:1.2.3"},
			{desc: "empty tag", input: "docker.io/mirantis/ucp:"},
			{desc: "empty repository", input: "/ucp:1.2.3"},
			{desc: "shell metacharacter in tag", input: "docker.io/mirantis/ucp:3.7.0;id"},
		}

		for _, tc := range cases {
			t.Run(tc.desc, func(t *testing.T) {
				image, err := docker.NewImage(tc.input)
				require.Error(t, err)
				require.Nil(t, image)
				require.Contains(t, err.Error(), tc.input)
			})
		}
	})
}

func TestAllFromString(t *testing.T) {
	images := docker.AllFromString(`docker.io/foo/bar:1.2.3
docker.io/foo/bar2:1.2.3`)
	require.Equal(t, 2, len(images))
	require.Equal(t, "docker.io/foo", images[0].Repository)
	require.Equal(t, "docker.io/foo/bar:1.2.3", images[0].String())
	require.Equal(t, "docker.io/foo/bar2:1.2.3", images[1].String())
}

func TestAllToRepository(t *testing.T) {
	images := docker.AllFromString(`docker.io/foo/bar:1.2.3
docker.io/foo/bar2:1.2.3-tp1  
docker.io:8888/foo2/bar3:PRODENG

`) // there is whitespace EOL on purpose

	require.Equal(t, 3, len(images))
	require.Equal(t, "docker.io/foo/bar:1.2.3", images[0].String())
	require.Equal(t, "docker.io/foo/bar2:1.2.3-tp1", images[1].String())
	require.Equal(t, "docker.io:8888/foo2/bar3:PRODENG", images[2].String())

	moved := docker.AllToRepository(images, "custom.example.com/repo")
	require.Equal(t, 3, len(moved))
	require.Equal(t, "custom.example.com/repo/bar:1.2.3", moved[0].String())
	require.Equal(t, "custom.example.com/repo/bar2:1.2.3-tp1", moved[1].String())
	require.Equal(t, "custom.example.com/repo/bar3:PRODENG", moved[2].String())
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
time="2025-07-03T01:00:12Z" level=info msg="Bootsrapper image org: registry.ci.mirantis.com/mirantiseng"
time="2025-07-03T01:00:12Z" level=info msg="Bootsrapper image version: 3.8.7"
registry.ci.mirantis.com/mirantiseng/ucp-agent:3.8.7
registry.ci.mirantis.com/mirantiseng/ucp-alertmanager:3.8.7
registry.ci.mirantis.com/mirantiseng/ucp-auth-store:3.8.7`
	images := docker.AllFromString(mke387output)
	require.Equal(t, 3, len(images))
	require.Equal(t, "registry.ci.mirantis.com/mirantiseng/ucp-agent:3.8.7", images[0].String())
	require.Equal(t, "registry.ci.mirantis.com/mirantiseng/ucp-alertmanager:3.8.7", images[1].String())
	require.Equal(t, "registry.ci.mirantis.com/mirantiseng/ucp-auth-store:3.8.7", images[2].String())
}

func TestImageVersions(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		tag      string
	}{
		{"mirantis/ucp-proxy:3.8.8", "mirantis/ucp-proxy", "3.8.8"},
		{"dtr.efzp.com:9026/mirantis/ucp-agent:3.8.10", "dtr.efzp.com:9026/mirantis/ucp-agent", "3.8.10"},
		{"ucp-proxy:3.8.8", "ucp-proxy", "3.8.8"},
		{"docker.io/library/alpine:latest", "docker.io/library/alpine", "latest"},
		{"localhost:5000/my-image:1.0.0", "localhost:5000/my-image", "1.0.0"},
		{"registry.mirantis.com/mirantiseng/ucp:3.9.0-rc2", "registry.mirantis.com/mirantiseng/ucp", "3.9.0-rc2"},
		{"registry.ci.mirantis.com/mirantiseng/ecp", "registry.ci.mirantis.com/mirantiseng/ecp", "latest"},
		{"localhost:5000/my-image", "localhost:5000/my-image", "latest"},
		{"my-complex-repo:5000/org/image:v1", "my-complex-repo:5000/org/image", "v1"},
	}

	for _, tc := range cases {
		repo, tag, err := docker.ImageRepoAndTag(tc.input)
		require.Nil(t, err, "ImageRepoAndTag gave unexpected error from valid version string: %s", tc.input)
		require.Equal(t, tc.expected, repo, "ImageRepoAndTag gave wrong repo value for: %s", tc.input)
		require.Equal(t, tc.tag, tag, "ImageRepoAndTag gave wrong tag value for: %s", tc.input)
	}
}

func TestImageVersionsWithColon(t *testing.T) {
	repo, tag, err := docker.ImageRepoAndTag("dtr.efzp.com:9026/mirantis/ucp-agent:3.8.10")
	require.Nil(t, err, "SwarmMKEVersion gave unexpected error from valid version string")
	require.Equal(t, "dtr.efzp.com:9026/mirantis/ucp-agent", repo, "SwarmMKEVersion gave wrong repo value")
	require.Equal(t, "3.8.10", tag, "SwarmMKEVersion gave wrong repo value")
}
