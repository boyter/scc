// SPDX-License-Identifier: MIT

package processor

import "testing"

func TestEnvSplitStringShebang(t *testing.T) {
	ProcessConstants()
	for _, tc := range []struct{ input, want string }{
		{"#!/usr/bin/env -S python3 -u\n", "Python"},
		{"#!/usr/bin/env -Spython3 -u\n", "Python"},
		{"#!/usr/bin/env --split-string=python3 -u\n", "Python"},
		{"#!/usr/bin/env -S FOO=bar python3 -u\n", "Python"},
		{"#!/usr/bin/env -S node --flag\n", "JavaScript"},
		{"#!/usr/bin/env python3\n", "Python"},
		{"#!/usr/bin/python3 -u\n", "Python"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := DetectSheBang([]byte(tc.input))
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestEnvSplitStringUnsupported(t *testing.T) {
	ProcessConstants()
	for _, input := range []string{
		"#!/usr/bin/env -S\n",
		"#!/usr/bin/env -S 'python3' -u\n",
		"#!/usr/bin/env --split-string python3 -u\n",
		"#!/usr/bin/env -S --bad=flag python3\n",
		"#!/usr/bin/env -S python3\u00a0-u\n",
	} {
		if got, err := DetectSheBang([]byte(input)); err != errUnknownShebang || got != "" {
			t.Errorf("%q: got %q, %v; want unknown", input, got, err)
		}
	}
}
