package processor

import (
	"bytes"
)

// this engine deals with any language that has no complexity, no comments and no quotes such as Plain Text, JSON
// Markdown etc...
// As such all it does is count the number of newlines and then check if any of those newlines are next to each
// other allowing us to count as a blank line which is far faster than the state machine used for languages


type TextEngine struct {}

func NewTextEngine() TextEngine {
	return TextEngine{}
}

func (c *TextEngine) Process(fileJob *FileJob) {
	locs := indexAll(fileJob.Content, []byte{'\n'})

	if len(locs) == 0 {
		fileJob.Lines = 1
	}

	fileJob.Lines = int64(len(locs)) + 1

	for _, b := range locs {
		// if there is a newline as the first character then we have 1 blank line
		if b[0] == 0 {
			fileJob.Blank++
		} else {
			// if we arent at the start then check if the previous is a newline and if so add another blank
			if b[0] != 0 && fileJob.Content[b[0]-1] == '\n' {
				fileJob.Blank++
			} else {
				// if the previous is \r check to see if its at the start in which case increment
				if fileJob.Content[b[0]-1] == '\r' && b[0]-1 == 0 {
					fileJob.Blank++
				} else if b[0]-2 > 0 && fileJob.Content[b[0]-2] == '\n' {
					fileJob.Blank++
				}
			}
		}
	}
}


// IndexAll extracts all of the locations of a string inside another string
// without regular expressions  which makes it faster than regex FindAllIndex in most
// situations while not being any slower. It performs worst when working against random
// data.
//
// Note it is not a drop in replacement for FindAllIndex because we don't do any limit check
func indexAll(haystack []byte, needle []byte) [][]int {
	// The below needed to avoid timeout crash found using go-fuzz
	if len(haystack) == 0 || len(needle) == 0 {
		return nil
	}

	// Return contains a slice of slices where index 0 is the location of the match in bytes
	// and index 1 contains the end location in bytes of the match
	var locs [][]int

	// Perform the first search outside the main loop to make the method
	// easier to understand
	searchText := haystack
	offSet := 0
	loc := bytes.Index(searchText, needle)

	for loc != -1 {
		// trim off the portion we already searched, and look from there
		searchText = searchText[loc+len(needle):]
		locs = append(locs, []int{loc + offSet, loc + offSet + len(needle)})

		// We need to keep the offset of the match so we continue searching
		offSet += loc + len(needle)

		// strings.Index does checks of if the string is empty so we don't need
		// to explicitly do it ourselves
		loc = bytes.Index(searchText, needle)
	}

	return locs
}
