package gocodewalker

import (
	"regexp"
	"strings"
)

func extractGitModuleFolders(input string) []string {
	// Compile a regular expression to match lines starting with "path ="
	re := regexp.MustCompile(`^\s*path\s*=\s*(.*)`)
	output := []string{}

	for _, line := range strings.Split(input, "\n") {
		// Check if the line matches the "path = " pattern
		if matches := re.FindStringSubmatch(line); matches != nil {
			// Extract the submodule path (which is captured in the regex group)
			submodulePath := strings.TrimSpace(matches[1])
			output = append(output, submodulePath)
		}
	}

	return output
}
