package processor

type StateBlank struct{}

func (state *StateBlank) String() string {
	return "blank"
}

func (state *StateBlank) Process(job *FileJob, lang *LanguageFeature, index int, lineType LineType) (int, LineType, State) {
	switch tokenType, offsetJump, endString := lang.Tokens.Match(job.Content[index:]); tokenType {
	case TMlcomment:
		commentType := lineType
		if commentType == LINE_BLANK {
			commentType = LINE_COMMENT
		}

		index += offsetJump - 1
		return index, commentType, NewStateCommentMulti(endString)

	case TSlcomment:
		commentType := lineType
		if commentType == LINE_BLANK {
			commentType = LINE_COMMENT
		}
		return index, commentType, &StateCommentSingle{}

	case TString:
		index, docString, skipEsc := verifyIgnoreEscape(lang, job, index)

		if docString {
			commentType := lineType
			if commentType == LINE_BLANK {
				commentType = LINE_COMMENT
			}

			return index, commentType, &StateDocString{
				End:     endString,
				SkipEsc: skipEsc,
			}
		}

		return index, LINE_CODE, &StateString{
			End:     endString,
			SkipEsc: skipEsc,
		}

	case TComplexity:
		if index == 0 || isWhitespace(job.Content[index-1]) {
			job.Complexity++
		}
		return index, LINE_BLANK, state

	default:
		return index, LINE_CODE, &StateCode{}
	}
}

func (state *StateBlank) Reset() (LineType, State) {
	return LINE_BLANK, state
}
