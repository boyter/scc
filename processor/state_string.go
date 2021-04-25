package processor

import "fmt"

type StateString struct {
	End     []byte
	SkipEsc bool
}

func (state *StateString) String() string {
	return fmt.Sprintf("string[end=%s,skipesc=%v]", state.End, state.SkipEsc)
}

func (state *StateString) Process(job *FileJob, lang *LanguageFeature, index int, lineType LineType) (int, LineType, State) {
	var i int
	for i = index; i < job.EndPoint; i++ {
		// If we hit a newline, return because we want to count the stats but keep
		// the current state so we end up back in this loop when the outer
		// one calls again
		if job.Content[i] == '\n' {
			return i, LINE_CODE, state
		}

		// If we are in a literal string we want to ignore the \ check OR we aren't checking for special ones
		if state.SkipEsc || judgeEscape(i, job) {
			if checkForMatchSingle(job.Content[i], i, job.EndPoint, state.End, job) {
				return i, LINE_CODE, &StateCode{}
			}
		}
	}

	return i, LINE_CODE, state
}

func (state *StateString) Reset() (LineType, State) {
	return LINE_CODE, state
}

// judge if the slash count before index is even number
func judgeEscape(index int, fileJob *FileJob) bool {
	slashCount := 0
	i := 1
	for index >= i && fileJob.Content[index-i] == '\\' {
		slashCount++
		i++
	}
	return slashCount%2 == 0
}
