package processor

type StateCode struct{}

func (state *StateCode) String() string {
	return "code"
}

func (state *StateCode) Process(job *FileJob, lang *LanguageFeature, index int, lineType LineType) (int, LineType, State) {
	// Hacky fix to https://github.com/boyter/scc/issues/181
	endPoint := job.EndPoint
	if endPoint > len(job.Content) {
		endPoint--
	}

	var i int
	for i = index; i < endPoint; i++ {
		curByte := job.Content[i]

		if curByte == '\n' {
			return i, LINE_CODE, state
		}

		if isBinary(i, curByte) {
			job.Binary = true
			return i, LINE_CODE, state
		}

		if shouldProcess(curByte, lang.ProcessMask) {
			if Duplicates {
				// Technically this is wrong because we skip bytes so this is not a true
				// hash of the file contents, but for duplicate files it shouldn't matter
				// as both will skip the same way
				digestible := []byte{job.Content[index]}
				job.Hash.Write(digestible)
			}

			switch tokenType, offsetJump, endString := lang.Tokens.Match(job.Content[i:]); tokenType {
			case TString:
				// If we are in string state then check what sort of string so we know if docstring OR ignoreescape string

				// It is safe to -1 here as to enter the code state we need to have
				// transitioned from blank to here hence i should always be >= 1
				// This check is to ensure we aren't in a character declaration
				// TODO this should use language features
				if job.Content[i-1] == '\\' {
					break // from switch, not from the loop
				}

				i, docString, skipEsc := verifyIgnoreEscape(lang, job, i)

				if docString {
					commentType := lineType
					if commentType == LINE_BLANK {
						commentType = LINE_COMMENT
					}

					return i, commentType, &StateDocString{
						End:     endString,
						SkipEsc: skipEsc,
					}
				}

				// i += offsetJump - 1
				return i, LINE_CODE, &StateString{
					End:     endString,
					SkipEsc: skipEsc,
				}

			case TSlcomment:
				i += offsetJump - 1
				return i, LINE_CODE, &StateCommentSingle{}

			case TMlcomment:
				i += offsetJump - 1

				return i, LINE_CODE, NewStateCommentMulti(endString)

			case TComplexity:
				if i == 0 || isWhitespace(job.Content[i-1]) {
					job.Complexity++
				}
			}
		}
	}

	return i, LINE_CODE, state
}

func (state *StateCode) Reset() (LineType, State) {
	return LINE_BLANK, &StateBlank{}
}
