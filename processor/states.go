package processor

type State interface {
	Process(*FileJob, *LanguageFeature, int, LineType) (int, LineType, State)
	Reset() (LineType, State)
}
