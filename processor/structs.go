package processor

type FileJob struct {
	Language   string
	Filename   string
	Extension  string
	Location   string
	Content    []byte
	Bytes      int64
	Lines      int64
	Code       int64
	Comment    int64
	Blank      int64
	Complexity int64
}

type LanguageSummary struct {
	Name       string
	Bytes      int64
	Lines      int64
	Code       int64
	Comment    int64
	Blank      int64
	Complexity int64
	Count      int64
}
