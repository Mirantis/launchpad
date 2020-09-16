package api

import (
	"strings"
)

// Flags is a slice of strings with added functions to ease manipulating lists of command-line flags
type Flags []string

func (f *Flags) Add(s string) {
	*f = append(*f, s)
}

// AddUnlessExist adds a flag unless one with the same prefix exists
func (f *Flags) AddUnlessExist(s string) {
	if f.Include(s) {
		return
	}
	*f = append(*f, s)
}

// AddOrReplace replaces a flag with the same prefix or adds a new one if one does not exist
func (f *Flags) AddOrReplace(s string) {
	idx := f.Index(s)
	if idx > -1 {
		(*f)[idx] = s
		return
	}
	*f = append(*f, s)
}

// Include returns true if a flag with a matching prefix can be found
func (f Flags) Include(s string) bool {
	return f.Index(s) > -1
}

// Index returns an index to a flag with a matching prefix
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

// Get returns the full flag with the possible value such as "--san=10.0.0.1" or "" when not found
func (f Flags) Get(s string) string {
	idx := f.Index(s)
	if idx < 0 {
		return ""
	}
	return f[idx]
}

// GetValue returns the value part of a flag such as "10.0.0.1" for a flag like "--san=10.0.0.1"
func (f Flags) GetValue(s string) string {
	fl := f.Get(s)
	if fl == "" {
		return ""
	}

	idx := strings.IndexAny(fl, "= ")
	if idx < 0 {
		return ""
	}

	return fl[idx+1:]
}

// Delete removes a matching flag from the list
func (f *Flags) Delete(s string) {
	idx := f.Index(s)
	if idx < 0 {
		return
	}
	*f = append((*f)[:idx], (*f)[idx+1:]...)
}
