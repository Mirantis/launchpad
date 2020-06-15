package v1beta1

// Hosts is a collection of Hosts
type Hosts []*Host

// Filter returns a filtered list of Hosts. The filter function should returnn true for hosts matching the criteria.
func (hosts *Hosts) Filter(filter func(host *Host) bool) Hosts {
	result := make([]*Host, 0, len(*hosts))

	for _, h := range *hosts {
		if filter(h) {
			result = append(result, h)
		}
	}

	return result
}

// Find returns the first matching Host. The finder function should return true for a Host matching the criteria.
func (hosts *Hosts) Find(filter func(host *Host) bool) *Host {
	for _, h := range *hosts {
		if filter(h) {
			return (h)
		}
	}
	return nil
}

// Index returns the index of the first matching Host. The finder function should return true for a Host matching the criteria.
func (hosts *Hosts) Index(filter func(host *Host) bool) int {
	for i, h := range *hosts {
		if filter(h) {
			return (i)
		}
	}
	return -1
}

// IndexAll returns the indexes of the matching Hosts. The finder function should return true for a Host matching the criteria.
func (hosts *Hosts) IndexAll(filter func(host *Host) bool) []int {
	result := make([]int, 0, len(*hosts))
	for i, h := range *hosts {
		if filter(h) {
			result = append(result, i)
		}
	}
	return result
}

// Map returns a new slice which is the result of running the map function on each host.
func (hosts *Hosts) Map(filter func(host *Host) interface{}) []interface{} {
	result := make([]interface{}, len(*hosts))
	for _, h := range *hosts {
		result = append(result, filter(h))
	}
	return result
}

// MapString returns a new slice which is the result of running the map function on each host
func (hosts *Hosts) MapString(filter func(host *Host) string) []string {
	result := make([]string, len(*hosts))
	for _, h := range *hosts {
		result = append(result, filter(h))
	}
	return result
}

// Include returns true if any of the hosts match the filter function criteria.
func (hosts *Hosts) Include(filter func(host *Host) bool) bool {
	for _, h := range *hosts {
		if filter(h) {
			return true
		}
	}
	return false
}

// Count returns the count of hosts matching the filter function criteria.
func (hosts *Hosts) Count(filter func(host *Host) bool) int {
	return len(hosts.IndexAll(filter))
}
