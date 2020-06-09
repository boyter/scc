package processor

import "testing"

func TestTextEngineLine(t *testing.T) {
	var cases = []struct{
		content string
		wantLines int64
	}{
		{"test", 1},
		{"test\nanother", 2},
		{"test\nanother\n", 3},
		{"\n", 2},
		{"\r\n", 2},
		{"test\r\nanother", 2},
	}

	for i, c := range cases {
		e := NewTextEngine()

		res := &FileJob{
			Content: []byte(c.content),
		}

		e.Process(res)

		if res.Lines != c.wantLines {
			t.Error("for", i, "expected", c.wantLines, "got", res.Lines)
		}
	}
}


func TestTextEngineBlank(t *testing.T) {
	var cases = []struct{
		content string
		wantBlank int64
	}{
		{"test", 0},
		{"test\nanother", 0},
		{"test\nanother\n", 0},
		{"\n", 1},
		{"\r\n", 1},
		{"test\n\nanother", 1},
		{"test\n\n\nanother", 2},
	}

	for i, c := range cases {
		e := NewTextEngine()

		res := &FileJob{
			Content: []byte(c.content),
		}

		e.Process(res)

		if res.Blank != c.wantBlank {
			t.Error("for", i, "expected", c.wantBlank, "got", res.Blank)
		}
	}
}
