package config

import (
	"strings"

	"github.com/Mirantis/launchpad/pkg/util/shellword"
)

// Flags is a slice of strings with added functions to ease manipulating lists of command-line flags.
type Flags []string

// Add adds a flag regardless if it exists already or not.
func (f *Flags) Add(s string) {
	*f = append(*f, s)
}

// AddUnlessExist adds a flag unless one with the same prefix exists.
func (f *Flags) AddUnlessExist(s string) {
	if f.Include(s) {
		return
	}
	*f = append(*f, s)
}

// AddOrReplace replaces a flag with the same prefix or adds a new one if one does not exist.
func (f *Flags) AddOrReplace(s string) {
	idx := f.Index(s)
	if idx > -1 {
		(*f)[idx] = s
		return
	}
	*f = append(*f, s)
}

// Include returns true if a flag with a matching prefix can be found.
func (f Flags) Include(s string) bool {
	return f.Index(s) > -1
}

// Index returns an index to a flag with a matching prefix.
func (f Flags) Index(s string) int {
	var flag string
	sepidx := strings.IndexAny(s, "= ")
	if sepidx < 0 {
		flag = s
	} else {
		flag = s[:sepidx]
	}
	for i, v := range f {
		if v == s || strings.HasPrefix(v, flag+"=") || strings.HasPrefix(v, flag+" ") {
			return i
		}
	}
	return -1
}

// Get returns the full flag with the possible value such as "--san=10.0.0.1" or "" when not found.
func (f Flags) Get(s string) string {
	idx := f.Index(s)
	if idx < 0 {
		return ""
	}
	return f[idx]
}

// GetValue returns the value part of a flag such as "10.0.0.1" for a flag like "--san=10.0.0.1".
func (f Flags) GetValue(s string) string {
	fl := f.Get(s)
	return FlagValue(fl)
}

// GetValues returns the value parts of every flag matching the prefix, such as
// ["10.0.0.0/8", "192.168.0.0/16"] for a repeated flag like
// "--default-addr-pool=10.0.0.0/8 --default-addr-pool=192.168.0.0/16". Returns
// nil when no flag matches.
func (f Flags) GetValues(s string) []string {
	var flag string
	sepidx := strings.IndexAny(s, "= ")
	if sepidx < 0 {
		flag = s
	} else {
		flag = s[:sepidx]
	}
	var values []string
	for _, v := range f {
		if v == s || strings.HasPrefix(v, flag+"=") || strings.HasPrefix(v, flag+" ") {
			values = append(values, FlagValue(v))
		}
	}
	return values
}

// Delete removes a matching flag from the list.
func (f *Flags) Delete(s string) {
	idx := f.Index(s)
	if idx < 0 {
		return
	}
	*f = append((*f)[:idx], (*f)[idx+1:]...)
}

// Merge takes the flags from another Flags and adds them to this one unless this already has that flag set.
func (f *Flags) Merge(b Flags) {
	for _, flag := range b {
		f.AddUnlessExist(flag)
	}
}

// MergeOverwrite takes the flags from another Flags and adds or replaces them into this one.
func (f *Flags) MergeOverwrite(b Flags) {
	for _, flag := range b {
		f.AddOrReplace(flag)
	}
}

// MergeAdd takes the flags from another Flags and adds them into this one even if they exist.
func (f *Flags) MergeAdd(b Flags) {
	for _, flag := range b {
		f.Add(flag)
	}
}

// Join creates a string separated by spaces.
func (f *Flags) Join() string {
	return strings.Join(*f, " ")
}

// FlagValue returns the value part of a string such as "--san 10.0.0.1" or "--san=10.0.0.1".
//
// The value is decoded as a POSIX shell would, which is how every consumer of
// these flags reads them: a flag list is joined and handed to a remote shell,
// and the credential flags are decoded by pkg/util/secretenv before being
// staged as environment values. This used to use strconv.Unquote, which reads
// Go string rules instead -- "C:\temp" came back as C:<TAB>emp and a
// single-quoted value kept its quotes -- so the same configured credential
// decoded one way for the MSR install and another for the MSR destroy.
func FlagValue(s string) string {
	if s == "" {
		return ""
	}

	idx := strings.IndexAny(s, "= ")
	if idx < 0 {
		return ""
	}

	return shellword.Unquote(s[idx+1:])
}
