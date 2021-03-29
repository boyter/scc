package processor

type StateCommentSingle struct {}

func (state *StateCommentSingle) String() string {
	return "comment"
}

func (state *StateCommentSingle) Process(job *FileJob, lang *LanguageFeature, index int, lineType LineType) (int, LineType, State) {
	var i int
	for i = index; i < job.EndPoint; i++ {
		curByte := job.Content[i]

		if curByte == '\n' {
			break
		}
	}

	return i, lineType, state
}

func (state *StateCommentSingle) Reset() (LineType, State) {
	return LINE_BLANK, &StateBlank{}
}
