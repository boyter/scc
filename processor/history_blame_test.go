// SPDX-License-Identifier: MIT

package processor

import (
	"strings"
	"testing"
)

func TestApplyDiffToBlameNewFile(t *testing.T) {
	got := applyDiffToBlame(nil, 3, []LineRange{{Start: 1, Count: 3}}, nil, 7)
	want := []authorID{7, 7, 7}
	if !equalIDs(got, want) {
		t.Errorf("new file blame = %v, want %v", got, want)
	}
}

func TestApplyDiffToBlameAppend(t *testing.T) {
	prev := []authorID{1, 1}
	got := applyDiffToBlame(prev, 4, []LineRange{{Start: 3, Count: 2}}, nil, 9)
	want := []authorID{1, 1, 9, 9}
	if !equalIDs(got, want) {
		t.Errorf("append blame = %v, want %v", got, want)
	}
}

func TestApplyDiffToBlamePureDelete(t *testing.T) {
	prev := []authorID{1, 1, 1, 1}
	got := applyDiffToBlame(prev, 2, nil, []LineRange{{Start: 2, Count: 2}}, 9)
	want := []authorID{1, 1}
	if !equalIDs(got, want) {
		t.Errorf("delete blame = %v, want %v", got, want)
	}
}

func TestApplyDiffToBlameReplaceMiddle(t *testing.T) {
	// 5 lines from author 1; replace line 3 with two lines from author 2.
	prev := []authorID{1, 1, 1, 1, 1}
	got := applyDiffToBlame(prev, 6,
		[]LineRange{{Start: 3, Count: 2}},
		[]LineRange{{Start: 3, Count: 1}},
		2)
	want := []authorID{1, 1, 2, 2, 1, 1}
	if !equalIDs(got, want) {
		t.Errorf("replace blame = %v, want %v", got, want)
	}
}

func TestApplyDiffToBlamePadsToNewLines(t *testing.T) {
	// Diff arithmetic disagrees with newLines — defensive pad with sentinel.
	got := applyDiffToBlame(nil, 4, nil, nil, 9)
	want := []authorID{0, 0, 0, 0}
	if !equalIDs(got, want) {
		t.Errorf("pad blame = %v, want %v", got, want)
	}
}

func TestAuthorRegistryInternsCanonical(t *testing.T) {
	mm := parseMailmap([]byte("Alice <alice@example.com> <a@example.com>\n"))
	r := newAuthorRegistry(mm)
	id1 := r.intern("Alice", "a@example.com")
	id2 := r.intern("Alice", "alice@example.com")
	if id1 != id2 {
		t.Errorf("mailmap-folded identities should collapse: %d vs %d", id1, id2)
	}
	if r.record(id1).Email != "alice@example.com" {
		t.Errorf("canonical email = %q, want alice@example.com", r.record(id1).Email)
	}
}

func TestAuthorRegistrySentinelReserved(t *testing.T) {
	r := newAuthorRegistry(nil)
	id := r.intern("Bob", "bob@example.com")
	if id == sentinelAuthorID {
		t.Errorf("real author should not be assigned sentinel ID")
	}
}

func TestParseMailmapNameOnly(t *testing.T) {
	m := parseMailmap([]byte("Proper Name <commit@example.com>\n"))
	name, email := m.Resolve("Other Name", "commit@example.com")
	if name != "Proper Name" {
		t.Errorf("name = %q, want Proper Name", name)
	}
	if email != "commit@example.com" {
		t.Errorf("email = %q, want commit@example.com", email)
	}
}

func TestParseMailmapEmailReplacement(t *testing.T) {
	m := parseMailmap([]byte("<proper@example.com> <commit@example.com>\n"))
	name, email := m.Resolve("Commit Name", "commit@example.com")
	if email != "proper@example.com" {
		t.Errorf("email = %q, want proper@example.com", email)
	}
	if name != "Commit Name" {
		t.Errorf("name = %q, want unchanged Commit Name", name)
	}
}

func TestParseMailmapNameAndEmailReplacement(t *testing.T) {
	m := parseMailmap([]byte("Proper <proper@example.com> Commit <commit@example.com>\n"))
	// Should only match when commit name AND commit email both match.
	name, email := m.Resolve("Commit", "commit@example.com")
	if name != "Proper" || email != "proper@example.com" {
		t.Errorf("got (%q,%q), want (Proper, proper@example.com)", name, email)
	}
	// Different commit name → no match.
	name2, email2 := m.Resolve("Other", "commit@example.com")
	if name2 != "Other" || email2 != "commit@example.com" {
		t.Errorf("got (%q,%q), want unchanged", name2, email2)
	}
}

func TestParseMailmapSkipsCommentsAndBlanks(t *testing.T) {
	body := "# comment\n\nAlice <a@x>  # trailing comment\n"
	m := parseMailmap([]byte(body))
	if len(m.byEmail) != 1 {
		t.Errorf("byEmail entries = %d, want 1", len(m.byEmail))
	}
}

func TestMailmapResolveNilSafe(t *testing.T) {
	var m *mailmap
	n, e := m.Resolve("Bob", "b@x")
	if n != "Bob" || e != "b@x" {
		t.Errorf("nil mailmap should be no-op, got (%q,%q)", n, e)
	}
}

func TestParseMailmapLineForms(t *testing.T) {
	cases := []struct {
		in      string
		properN string
		properE string
		commitN string
		commitE string
		ok      bool
	}{
		{"Proper Name <c@x>", "Proper Name", "", "", "c@x", true},
		{"<p@x> <c@x>", "", "p@x", "", "c@x", true},
		{"Proper Name <p@x> <c@x>", "Proper Name", "p@x", "", "c@x", true},
		{"Proper Name <p@x> Commit Name <c@x>", "Proper Name", "p@x", "Commit Name", "c@x", true},
		{"no brackets here", "", "", "", "", false},
	}
	for _, c := range cases {
		got, ok := parseMailmapLine(c.in)
		if ok != c.ok {
			t.Errorf("parseMailmapLine(%q) ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if got.properName != c.properN || got.properEmail != c.properE ||
			got.commitName != c.commitN || got.commitEmail != c.commitE {
			t.Errorf("parseMailmapLine(%q) = %+v, want (%q,%q,%q,%q)",
				c.in, got, c.properN, c.properE, c.commitN, c.commitE)
		}
	}
}

func TestParseMailmapTrimsWhitespace(t *testing.T) {
	m := parseMailmap([]byte("  Proper Name   <c@x>  \n"))
	if e, ok := m.byEmail["c@x"]; !ok || e.Name != "Proper Name" {
		t.Errorf("byEmail[c@x] = %+v, want Proper Name", e)
	}
}

// TestAuthorRegistryFoldsByNameAndDomain documents the canonical scc-repo
// case: the same human committing from a personal address and a
// github-noreply address. The two have different domains so under the
// strict (name, domain) rule they intentionally stay split — folding here
// would require the looser name-only mode which is not the default.
func TestAuthorRegistryFoldsByNameAndDomain(t *testing.T) {
	r := newAuthorRegistryWithFold(nil, true)
	id1 := r.intern("Ben Boyter", "ben@boyter.org")
	id2 := r.intern("Ben Boyter", "boyter@users.noreply.github.com")
	if id1 == id2 {
		t.Errorf("different domains should not fold under strict (name,domain): %d == %d", id1, id2)
	}
}

func TestAuthorRegistryFoldsSameDomainDifferentEmail(t *testing.T) {
	r := newAuthorRegistryWithFold(nil, true)
	id1 := r.intern("Alice", "alice@x.com")
	id2 := r.intern("Alice", "alice.smith@x.com")
	if id1 != id2 {
		t.Errorf("same (name,domain) should fold: %d vs %d", id1, id2)
	}
}

func TestAuthorRegistryDoesNotFoldDifferentDomain(t *testing.T) {
	r := newAuthorRegistryWithFold(nil, true)
	id1 := r.intern("Daniel", "d@a.com")
	id2 := r.intern("Daniel", "d@b.com")
	if id1 == id2 {
		t.Errorf("different domains should stay split: %d == %d", id1, id2)
	}
}

func TestAuthorRegistrySkipsGenericNames(t *testing.T) {
	r := newAuthorRegistryWithFold(nil, true)
	id1 := r.intern("root", "root@host-a")
	id2 := r.intern("root", "root@host-b")
	if id1 == id2 {
		t.Errorf("generic name 'root' must not fold across hosts: %d == %d", id1, id2)
	}
	// Even within the same domain, the skip list keeps them split — the
	// name is too generic to assume identity.
	id3 := r.intern("root", "root@host-a")
	if id3 != id1 {
		t.Errorf("identical (name,email) for skipped names should still dedupe by primary key: %d vs %d", id3, id1)
	}
}

func TestAuthorRegistryFoldsBots(t *testing.T) {
	r := newAuthorRegistryWithFold(nil, true)
	id1 := r.intern("dependabot[bot]", "x@y.com")
	id2 := r.intern("dependabot[bot]", "x@y.com")
	if id1 != id2 {
		t.Errorf("identical bot identities should collapse: %d vs %d", id1, id2)
	}
}

func TestAuthorRegistryMailmapBeatsFold(t *testing.T) {
	// Mailmap maps A→C and B→D. Even though A and B share a name+domain
	// after the (unrelated) folding key would suggest, the mailmap routes
	// them to distinct canonical identities and they must stay split.
	mm := parseMailmap([]byte(
		"Carol <carol@example.com> Sam <a@example.com>\n" +
			"Dave <dave@example.com> Sam <b@example.com>\n"))
	r := newAuthorRegistryWithFold(mm, true)
	id1 := r.intern("Sam", "a@example.com")
	id2 := r.intern("Sam", "b@example.com")
	if id1 == id2 {
		t.Errorf("mailmap-distinct identities must not fold: %d == %d", id1, id2)
	}
}

func TestAuthorRegistryFoldDisabled(t *testing.T) {
	r := newAuthorRegistryWithFold(nil, false)
	id1 := r.intern("Alice", "alice@x.com")
	id2 := r.intern("Alice", "alice.smith@x.com")
	if id1 == id2 {
		t.Errorf("with fold disabled, distinct emails must stay split: %d == %d", id1, id2)
	}
}

func TestEmailDomain(t *testing.T) {
	cases := []struct{ in, want string }{
		{"a@b.com", "b.com"},
		{"A@B.COM", "b.com"},
		{"a@b@c.com", "c.com"},
		{"noatsign", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := emailDomain(c.in); got != c.want {
			t.Errorf("emailDomain(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func equalIDs(a, b []authorID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestParseMailmapHandlesCaseInsensitiveEmail confirms that mailmap email
// keys are folded to lowercase so commits using mixed-case emails still
// resolve. Real-world commits often have inconsistent casing.
func TestParseMailmapHandlesCaseInsensitiveEmail(t *testing.T) {
	m := parseMailmap([]byte("Alice <alice@example.com> <a@example.com>\n"))
	n, e := m.Resolve("Anyone", strings.ToUpper("a@example.com"))
	if n != "Alice" || e != "alice@example.com" {
		t.Errorf("case-insensitive lookup got (%q,%q), want (Alice, alice@example.com)", n, e)
	}
}
