package processor

import "sync"

type StatCounter struct {
	languageFeatures map[string]LanguageFeature
	languageFeaturesMutex sync.Mutex
}

func NewCountStats() StatCounter {
	return StatCounter{}
}

// Takes in the filejob and processes it according to whatever rules are configured
// for both this stat counter and the filejob IE language etc..
func (c *StatCounter) Process(fileJob *FileJob) {
}



//"Plain Text": {
//    "complexitychecks": [],
//    "extensions": [
//      "text",
//      "txt"
//    ],
//    "line_comment": [],
//    "multi_line": [],
//    "quotes": []
//  },
