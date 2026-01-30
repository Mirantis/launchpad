package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlags_Add(t *testing.T) {
	var f Flags
	f.Add("--foo")
	require.Len(t, f, 1)
	require.Equal(t, "--foo", f[0])
	f.Add("--bar")
	require.Len(t, f, 2)
	require.Equal(t, "--bar", f[1])
	// Add does not deduplicate
	f.Add("--foo")
	require.Len(t, f, 3)
}

func TestFlags_AddUnlessExist(t *testing.T) {
	f := Flags{"--san=10.0.0.1"}
	f.AddUnlessExist("--san=10.0.0.2")
	require.Len(t, f, 1, "should not add when prefix exists")
	require.Equal(t, "--san=10.0.0.1", f[0])
	f.AddUnlessExist("--help")
	require.Len(t, f, 2)
	require.True(t, f.Include("--help"))
}

func TestFlags_AddOrReplace(t *testing.T) {
	f := Flags{"--san=10.0.0.1", "--foo"}
	f.AddOrReplace("--san=10.0.0.2")
	require.Len(t, f, 2)
	require.Equal(t, "--san=10.0.0.2", f[0])
	f.AddOrReplace("--new")
	require.Len(t, f, 3)
	require.Equal(t, "--new", f[2])
}

func TestFlags_Include(t *testing.T) {
	f := Flags{"--san=10.0.0.1", "--ucp-insecure-tls"}
	require.True(t, f.Include("--san"))
	require.True(t, f.Include("--ucp-insecure-tls"))
	require.False(t, f.Include("--missing"))
	require.False(t, f.Include("--sa")) // prefix must match fully
}

func TestFlags_Index(t *testing.T) {
	f := Flags{"--admin-username=foofoo", "--san foo", "--ucp-insecure-tls", "--x=y"}
	require.Equal(t, 0, f.Index("--admin-username"))
	require.Equal(t, 1, f.Index("--san"))
	require.Equal(t, 2, f.Index("--ucp-insecure-tls"))
	require.Equal(t, 3, f.Index("--x"))
	require.Equal(t, -1, f.Index("--missing"))
	// Index matches by prefix with = or space
	require.Equal(t, 1, f.Index("--san"))
	require.Equal(t, 0, f.Index("--admin-username=other"))
}

func TestFlags_Get(t *testing.T) {
	f := Flags{"--san=10.0.0.1", "--foo bar"}
	require.Equal(t, "--san=10.0.0.1", f.Get("--san"))
	require.Equal(t, "--foo bar", f.Get("--foo"))
	require.Equal(t, "", f.Get("--missing"))
}

func TestFlags_GetValue(t *testing.T) {
	f := Flags{"--san=10.0.0.1", "--foo bar"}
	require.Equal(t, "10.0.0.1", f.GetValue("--san"))
	require.Equal(t, "bar", f.GetValue("--foo"))
	require.Equal(t, "", f.GetValue("--missing"))
}

func TestFlags_Delete(t *testing.T) {
	f := Flags{"--a", "--b", "--c"}
	f.Delete("--b")
	require.Equal(t, Flags{"--a", "--c"}, f)
	f.Delete("--missing")
	require.Equal(t, Flags{"--a", "--c"}, f)
	f.Delete("--a")
	require.Equal(t, Flags{"--c"}, f)
}

func TestFlags_Merge(t *testing.T) {
	f := &Flags{"--existing=1"}
	f.Merge(Flags{"--existing=2", "--new=3"})
	require.Equal(t, 2, len(*f))
	require.Equal(t, "--existing=1", (*f)[0])
	require.True(t, f.Include("--new"))
}

func TestFlags_MergeOverwrite(t *testing.T) {
	f := &Flags{"--san=10.0.0.1", "--foo"}
	f.MergeOverwrite(Flags{"--san=10.0.0.2", "--new"})
	require.Len(t, *f, 3)
	require.Equal(t, "--san=10.0.0.2", (*f)[0])
	require.True(t, f.Include("--new"))
}

func TestFlags_MergeAdd(t *testing.T) {
	f := &Flags{"--san=1"}
	f.MergeAdd(Flags{"--san=2", "--other"})
	require.Len(t, *f, 3)
	require.Equal(t, "--san=1", (*f)[0])
	require.Equal(t, "--san=2", (*f)[1])
	require.Equal(t, "--other", (*f)[2])
}

func TestFlags_Join(t *testing.T) {
	f := &Flags{"--help", "--setting=false"}
	require.Equal(t, "--help --setting=false", f.Join())
	f = &Flags{}
	require.Equal(t, "", f.Join())
	f = &Flags{"--only"}
	require.Equal(t, "--only", f.Join())
}

func TestFlagValue(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		expected string
	}{
		{"empty", "", ""},
		{"no separator", "--flag", ""},
		{"equals", "--san=10.0.0.1", "10.0.0.1"},
		{"space", "--san 10.0.0.1", "10.0.0.1"},
		{"quoted double", "--user \"foo bar\"", "foo bar"},
		// strconv.Unquote only unquotes Go double-quoted strings; single-quoted value is returned as-is
		{"equals single-quoted value", "--x='a b'", "'a b'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FlagValue(tt.in)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestFlags(t *testing.T) {
	flags := Flags{"--admin-username=foofoo", "--san foo", "--ucp-insecure-tls"}
	require.Equal(t, "--ucp-insecure-tls", flags[2])
	require.Equal(t, 0, flags.Index("--admin-username"))
	require.Equal(t, 1, flags.Index("--san"))
	require.Equal(t, 2, flags.Index("--ucp-insecure-tls"))
	require.True(t, flags.Include("--san"))

	flags.Delete("--san")
	require.Equal(t, 1, flags.Index("--ucp-insecure-tls"))
	require.False(t, flags.Include("--san"))

	flags.AddOrReplace("--san 10.0.0.1")
	require.Equal(t, 2, flags.Index("--san"))
	require.Equal(t, "--san 10.0.0.1", flags.Get("--san"))
	require.Equal(t, "10.0.0.1", flags.GetValue("--san"))
	require.Equal(t, "foofoo", flags.GetValue("--admin-username"))

	require.Len(t, flags, 3)
	flags.AddOrReplace("--admin-password=barbar")
	require.Equal(t, 3, flags.Index("--admin-password"))
	require.Equal(t, "barbar", flags.GetValue("--admin-password"))

	require.Len(t, flags, 4)
	flags.AddUnlessExist("--admin-password=borbor")
	require.Len(t, flags, 4)
	require.Equal(t, "barbar", flags.GetValue("--admin-password"))

	flags.AddUnlessExist("--help")
	require.Len(t, flags, 5)
	require.True(t, flags.Include("--help"))
}

func TestFlagsWithQuotes(t *testing.T) {
	flags := Flags{"--admin-username \"foofoo\"", "--admin-password=\"foobar\""}
	require.Equal(t, "foofoo", flags.GetValue("--admin-username"))
	require.Equal(t, "foobar", flags.GetValue("--admin-password"))
}

func TestString(t *testing.T) {
	flags := Flags{"--help", "--setting=false"}
	require.Equal(t, "--help --setting=false", flags.Join())
}
