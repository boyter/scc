package processor

import (
	"strings"
)

var extensionCache = map[string]string{}

func getExtension(name string) string {
	extension = strings.ToLower(name)
	extension, ok := extensionCache[name]

	if ok {
		return extension
	}

	loc := strings.LastIndex(extension, ".")

	if loc != -1 {
		extension = extension[loc+1:]
	}

	extensionCache[name] = extension
	return extension
}
