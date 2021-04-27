package processor

type StateCommentMulti struct {
	Stack [][]byte
}

func (state *StateCommentMulti) String() string {
	return "multiline-comment"
}

func NewStateCommentMulti(token []byte) *StateCommentMulti {
	return &StateCommentMulti{
		Stack: [][]byte{token},
	}
}

func (state *StateCommentMulti) Process(job *FileJob, lang *LanguageFeature, index int, lineType LineType) (int, LineType, State) {
	var i int
	for i = index; i < job.EndPoint; i++ {
		curByte := job.Content[i]

		if curByte == '\n' {
			break
		}

		endToken := state.peek()
		if checkForMatchSingle(curByte, i, job.EndPoint, endToken, job) {
			// set offset jump here
			i += len(endToken) - 1

			if len(state.Stack) == 1 {
				return i, lineType, &StateBlank{}
			} else {
				state.pop()
				return i, lineType, state
			}
		}

		// Check if we are entering another multiline comment
		// This should come below check for match single as it speeds up processing
		if lang.Nested {
			if ok, offsetJump, endString := lang.MultiLineComments.Match(job.Content[i:]); ok != 0 {
				i += offsetJump - 1
				state.push(endString)
				return i, lineType, state
			}
		}
	}

	return i, lineType, state
}

func (state *StateCommentMulti) Reset() (LineType, State) {
	return LINE_COMMENT, state
}

func (state *StateCommentMulti) peek() []byte {
	i := len(state.Stack) - 1
	return state.Stack[i]
}

func (state *StateCommentMulti) push(token []byte) {
	state.Stack = append(state.Stack, token)
}

func (state *StateCommentMulti) pop() {
	i := len(state.Stack) - 1

	state.Stack = state.Stack[:i]
}
