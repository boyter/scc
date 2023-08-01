package main

import (
	"fmt"
	"math"
	"testing"
)

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

func Test_formatCount(t *testing.T) {

	fmt.Println(math.Round(2500 / 1000))

	type args struct {
		count float64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "",
			args: args{
				count: 100,
			},
			want: "100",
		},
		{
			name: "",
			args: args{
				count: 1000,
			},
			want: "1.0k",
		},
		{
			name: "",
			args: args{
				count: 2500,
			},
			want: "2.5k",
		},
		{
			name: "",
			args: args{
				count: 436465,
			},
			want: "436k",
		},
		{
			name: "",
			args: args{
				count: 263804,
			},
			want: "264k",
		},
		{
			name: "",
			args: args{
				count: 86400,
			},
			want: "86k",
		},
		{
			name: "",
			args: args{
				count: 81.99825581739397,
			},
			want: "82",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatCount(tt.args.count); got != tt.want {
				t.Errorf("formatCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
