package main

import "testing"

func Test_processPath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "",
			args: args{
				path: "/github/boyter/really-cheap-chatbot/",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := processPath(tt.args.path)
			if err != nil {
				t.Error("err")
			}
		})
	}
}
