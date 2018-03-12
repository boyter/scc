package processor

import (
	"strings"
)

var extensionCache = map[string]string{}

func getExtension(name string) string {
	name = strings.ToLower(name)
	extension, ok := extensionCache[name]

	if ok {
		return extension
	}

	loc := strings.LastIndex(name, ".")

	if loc != -1 {
		extension = name[loc+1:]
	} else {
		extension = name
	}

	extensionCache[name] = extension
	return extension
}
