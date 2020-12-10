package api

// Hooks define a list of hooks such as hooks["apply"]["before"] = ["ls -al", "rm foo.txt"]
type Hooks map[string]map[string][]string
