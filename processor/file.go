package processor

import (
	"strings"
)

var extensionCache = map[string]string{}

func getExtension(name string) string {
	extension, ok := extensionCache[name]

	if ok {
		return extension
	}

	extension = strings.ToLower(name)
	loc := strings.LastIndex(extension, ".")

	if loc == -1 {
		return extension
	} else {
		extension = extension[loc+1:]
	}

	extensionCache[name] = extension
	return extension
}
