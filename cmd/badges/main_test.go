package main

import (
	"net/url"
	"reflect"
	"testing"
)

func Test_resolveColor(t *testing.T) {
	tests := []struct {
		name  string
		color string
		want  string
	}{
		// Named colors (shields.io)
		{name: "blue", color: "blue", want: "007ec6"},
		{name: "red", color: "red", want: "e05d44"},
		{name: "green", color: "green", want: "97ca00"},
		{name: "brightgreen", color: "brightgreen", want: "44cc11"},
		{name: "yellowgreen", color: "yellowgreen", want: "a4a61d"},
		{name: "orange", color: "orange", want: "fe7d37"},
		{name: "yellow", color: "yellow", want: "dfb317"},
		{name: "lightgrey", color: "lightgrey", want: "9f9f9f"},
		{name: "lightgray", color: "lightgray", want: "9f9f9f"},
		{name: "blueviolet", color: "blueviolet", want: "8a2be2"},
		// Semantic aliases
		{name: "success", color: "success", want: "44cc11"},
		{name: "critical", color: "critical", want: "e05d44"},
		{name: "informational", color: "informational", want: "007ec6"},
		// CSS colors
		{name: "white", color: "white", want: "ffffff"},
		{name: "black", color: "black", want: "000000"},
		{name: "navy", color: "navy", want: "000080"},
		{name: "teal", color: "teal", want: "008080"},
		// Case insensitivity
		{name: "BLUE uppercase", color: "BLUE", want: "007ec6"},
		{name: "Blue mixed case", color: "Blue", want: "007ec6"},
		// Hex codes (3 digit)
		{name: "hex 3 digit", color: "fff", want: "fff"},
		{name: "hex 3 digit mixed", color: "a1b", want: "a1b"},
		// Hex codes (6 digit)
		{name: "hex 6 digit", color: "abcdef", want: "abcdef"},
		{name: "hex 6 digit uppercase", color: "ABCDEF", want: "abcdef"},
		// Hex codes (4 digit with alpha)
		{name: "hex 4 digit", color: "fffa", want: "fffa"},
		// Hex codes (8 digit with alpha)
		{name: "hex 8 digit", color: "abcdef12", want: "abcdef12"},
		// Invalid inputs
		{name: "invalid name", color: "notacolor", want: ""},
		{name: "invalid hex too short", color: "ab", want: ""},
		{name: "invalid hex too long", color: "abcdefghi", want: ""},
		{name: "invalid hex with special chars", color: "abc#ef", want: ""},
		{name: "empty string", color: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveColor(tt.color); got != tt.want {
				t.Errorf("resolveColor(%q) = %q, want %q", tt.color, got, tt.want)
			}
		})
	}
}

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
		{
			name: "named colors - shields.io style",
			values: url.Values{
				"font-color":              []string{"white"},
				"font-shadow-color":       []string{"black"},
				"top-shadow-accent-color": []string{"lightgrey"},
				"title-bg-color":          []string{"blue"},
				"badge-bg-color":          []string{"brightgreen"},
			},
			want: &badgeSettings{
				FontColor:            "ffffff",
				TextShadowColor:      "000000",
				TopShadowAccentColor: "9f9f9f",
				TitleBackgroundColor: "007ec6",
				BadgeBackgroundColor: "44cc11",
			},
		},
		{
			name: "mixed named colors and hex codes",
			values: url.Values{
				"font-color":              []string{"fff"},
				"font-shadow-color":       []string{"navy"},
				"top-shadow-accent-color": []string{"bbb"},
				"title-bg-color":          []string{"blueviolet"},
				"badge-bg-color":          []string{"success"},
			},
			want: &badgeSettings{
				FontColor:            "fff",
				TextShadowColor:      "000080",
				TopShadowAccentColor: "bbb",
				TitleBackgroundColor: "8a2be2",
				BadgeBackgroundColor: "44cc11",
			},
		},
		{
			name: "case insensitive named colors",
			values: url.Values{
				"font-color":     []string{"WHITE"},
				"title-bg-color": []string{"Blue"},
				"badge-bg-color": []string{"BrightGreen"},
			},
			want: &badgeSettings{
				FontColor:            "ffffff",
				TextShadowColor:      defaultSettings.TextShadowColor,
				TopShadowAccentColor: defaultSettings.TopShadowAccentColor,
				TitleBackgroundColor: "007ec6",
				BadgeBackgroundColor: "44cc11",
			},
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
