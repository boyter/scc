package processor

import (
	"testing"
)

var falseCases = map[string][]string{
	"":                    {""},
	"#a comment":          {"aaaa"},
	"!gitglob.go":         {"gitglob.go"},
	"/hello/**/world.txt": {"/some/dirs/foo.txt"},
}

func TestNewIgnoreMatcherFalseCases(t *testing.T) {
	for k, v := range falseCases {
		ig := NewIgnoreMatcher(k)

		for _, val := range v {
			if ig.Match(val) {
				t.Error("expected", k, "to be false for", val, "but was true")
			}
		}
	}
}

var trueCases = map[string][]string{
	"gitglob.go":          {"gitglob.go"},
	"/foo.txt":            {"/foo.txt"},
	"/hello/**/world.txt": {"/hello/world.txt", "/hello/stuff/world.txt"},
}

func TestNewIgnoreMatcherTrueCases(t *testing.T) {
	for k, v := range trueCases {
		ig := NewIgnoreMatcher(k)

		for _, val := range v {
			if !ig.Match(val) {
				t.Error("expected", k, "to be true for", val, "but was false")
			}
		}
	}
}
