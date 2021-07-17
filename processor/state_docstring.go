package processor

import (
	"fmt"
)

type StateDocString struct {
	End     []byte
	SkipEsc bool
}

func (state *StateDocString) String() string {
	return "docstring"
}

func (state *StateDocString) Process(job *FileJob, lang *LanguageFeature, index int, lineType LineType) (int, LineType, State) {
	var i int
	for i = index; i < job.EndPoint; i++ {
		if job.Content[i] == '\n' {
			return i, lineType, state
		}

		if job.Content[i-1] != '\\' {
			if checkForMatchSingle(job.Content[i], i, job.EndPoint, state.End, job) {
				// So we have hit end of docstring at this point in which case check if only whitespace characters till the next
				// newline and if so we change to a comment otherwise to code
				// need to start the loop after ending definition of docstring, therefore adding the length of the string to
				// the index
				for j := i + len(state.End); j <= job.EndPoint; j++ {
					if job.Content[j] == '\n' {
						if Debug {
							printDebug("Found newline so docstring is comment")
						}
						return j, LINE_COMMENT, &StateBlank{}
					}

					if !isWhitespace(job.Content[j]) {
						if Debug {
							printDebug(fmt.Sprintf("Found something not whitespace so is code: %s", string(job.Content[j])))
						}
						return j, LINE_CODE, &StateBlank{}
					}
				}
			}
		}
	}

	return i, lineType, state
}

func (state *StateDocString) Reset() (LineType, State) {
	return LINE_COMMENT, state
}
