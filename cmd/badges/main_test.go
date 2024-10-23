package main

import (
	"net/url"
	"reflect"
	"testing"
)

func Test_formatCount(t *testing.T) {
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
			want: "263k",
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

func Test_processPath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    location
		wantErr bool
	}{
		{
			name: "",
			args: args{
				path: "/github/boyter/really-cheap-chatbot/",
			},
			want: location{
				Provider: "github",
				User:     "boyter",
				Repo:     "really-cheap-chatbot",
			},
			wantErr: false,
		},
		{
			name: "",
			args: args{
				path: "github/boyter/really-cheap-chatbot",
			},
			want: location{
				Provider: "github",
				User:     "boyter",
				Repo:     "really-cheap-chatbot",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processUrlPath(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("processUrlPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processUrlPath() got = %v, want %v", got, tt.want)
			}
		})
	}
}
func Test_parseBadgeSettings(t *testing.T) {

	defaultSettings := &badgeSettings{
		FontColor:            "fff",
		TextShadowColor:      "010101",
		TopShadowAccentColor: "bbb",
		TitleBackgroundColor: "555",
		BadgeBackgroundColor: "4c1",
	}

	tests := []struct {
		name   string
		values url.Values
		want   *badgeSettings
	}{
		{
			name:   "default settings",
			values: url.Values{},
			want:   defaultSettings,
		},
		{
			name: "valid custom settings",
			values: url.Values{
				"font-color":              []string{"abcdef"},
				"font-shadow-color":       []string{"def"},
				"top-shadow-accent-color": []string{"321def"},
				"title-bg-color":          []string{"456"},
				"badge-bg-color":          []string{"789"},
			},
			want: &badgeSettings{
				FontColor:            "abcdef",
				TextShadowColor:      "def",
				TopShadowAccentColor: "321def",
				TitleBackgroundColor: "456",
				BadgeBackgroundColor: "789",
			},
		},
		{
			name: "partially-valid custom settings",
			values: url.Values{
				"font-color":              []string{"123321"},
				"font-shadow-color":       []string{"invalid"},
				"top-shadow-accent-color": []string{"5a534332"},
				"title-bg-color":          []string{"dd"},
				"badge-bg-color":          []string{"X&^%^#$^$@%20"},
			},
			want: &badgeSettings{
				FontColor:            "123321",
				TextShadowColor:      defaultSettings.TextShadowColor,
				TopShadowAccentColor: "5a534332",
				TitleBackgroundColor: defaultSettings.TitleBackgroundColor,
				BadgeBackgroundColor: defaultSettings.BadgeBackgroundColor,
			},
		},
		{
			name: "invalid custom settings",
			values: url.Values{
				"font-color":              []string{"invalid"},
				"font-shadow-color":       []string{"invalid"},
				"top-shadow-accent-color": []string{"invalid"},
				"title-bg-color":          []string{"invalid"},
				"badge-bg-color":          []string{"invalid"},
			},
			want: defaultSettings,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseBadgeSettings(tt.values); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseBadgeSettings() = %v, want %v", got, tt.want)
			}
		})
	}
}
