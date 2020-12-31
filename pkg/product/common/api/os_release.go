package api

// OsRelease host operating system info
type OsRelease struct {
	ID      string
	IDLike  string
	Name    string
	Version string
}

// String implements Stringer
func (o *OsRelease) String() string {
	return o.Name
}
