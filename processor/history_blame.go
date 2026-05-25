// SPDX-License-Identifier: MIT

package processor

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/object"
)

// authorID identifies an author after mailmap folding within a single
// observer instance. 0 is reserved for the "(before window)" sentinel that
// collects lines surviving from before the window's start commit.
type authorID uint32

const sentinelAuthorID authorID = 0

// authorRecord is the canonical (post-mailmap) identity stored against an
// authorID. The sentinel slot keeps zero values.
type authorRecord struct {
	Name  string
	Email string
}

// authorRegistry interns (name, email) pairs into stable authorIDs after
// folding them through the mailmap, if one is set. It is keyed by canonical
// name+email so two commit identities mapped to the same canonical pair
// collapse to one authorID.
type authorRegistry struct {
	nameToID map[string]authorID
	records  []authorRecord
	mm       *mailmap
}

func newAuthorRegistry(mm *mailmap) *authorRegistry {
	return &authorRegistry{
		nameToID: map[string]authorID{},
		records:  []authorRecord{{}}, // slot 0 = sentinelAuthorID
		mm:       mm,
	}
}

func (r *authorRegistry) intern(name, email string) authorID {
	canonName, canonEmail := name, email
	if r.mm != nil {
		canonName, canonEmail = r.mm.Resolve(canonName, canonEmail)
	}
	key := canonName + "\x00" + canonEmail
	if id, ok := r.nameToID[key]; ok {
		return id
	}
	id := authorID(len(r.records))
	r.records = append(r.records, authorRecord{Name: canonName, Email: canonEmail})
	r.nameToID[key] = id
	return id
}

func (r *authorRegistry) record(id authorID) authorRecord {
	if int(id) >= len(r.records) {
		return authorRecord{}
	}
	return r.records[id]
}

// applyDiffToBlame walks prev (the per-line author IDs from the previous
// commit) and the diff's added/removed line ranges in source order, returning
// the per-line author IDs for the new blob. Added lines are attributed to
// `commit`; equal lines copy from prev. The result is sized to newLines —
// truncated or sentinel-padded if the diff arithmetic disagrees with the
// classifier's line count (e.g. renames, trailing-newline differences).
func applyDiffToBlame(prev []authorID, newLines int, added, removed []LineRange, commit authorID) []authorID {
	out := make([]authorID, 0, newLines)
	oldPos := 1
	newPos := 1
	ai, ri := 0, 0
	oldN := len(prev)

	for newPos <= newLines || oldPos <= oldN {
		if ri < len(removed) && oldPos == removed[ri].Start {
			oldPos += removed[ri].Count
			ri++
			continue
		}
		if ai < len(added) && newPos == added[ai].Start {
			for k := 0; k < added[ai].Count; k++ {
				out = append(out, commit)
			}
			newPos += added[ai].Count
			ai++
			continue
		}
		if oldPos <= oldN && newPos <= newLines {
			out = append(out, prev[oldPos-1])
			oldPos++
			newPos++
			continue
		}
		break
	}

	if len(out) > newLines {
		out = out[:newLines]
	}
	for len(out) < newLines {
		out = append(out, sentinelAuthorID)
	}
	return out
}

// mailmap is a parsed .mailmap file. The standard line forms are:
//
//	Proper Name <commit@email>
//	<proper@email> <commit@email>
//	Proper Name <proper@email> <commit@email>
//	Proper Name <proper@email> Commit Name <commit@email>
//
// Lookup is by commit email, optionally also by commit name. The replacement
// fields override Name and/or Email; absent fields leave that part of the
// commit identity unchanged.
type mailmap struct {
	byEmail        map[string]mailmapEntry
	byNameAndEmail map[string]mailmapEntry
}

type mailmapEntry struct {
	Name  string
	Email string
}

// Resolve returns the canonical (name, email) after applying the mailmap.
// A nil receiver is a no-op so callers can intern unmapped identities the
// same way.
func (m *mailmap) Resolve(name, email string) (string, string) {
	if m == nil {
		return name, email
	}
	lookupEmail := strings.ToLower(email)
	if e, ok := m.byNameAndEmail[name+"\x00"+lookupEmail]; ok {
		return overrideIdentity(name, email, e)
	}
	if e, ok := m.byEmail[lookupEmail]; ok {
		return overrideIdentity(name, email, e)
	}
	return name, email
}

func overrideIdentity(name, email string, e mailmapEntry) (string, string) {
	outName, outEmail := name, email
	if e.Name != "" {
		outName = e.Name
	}
	if e.Email != "" {
		outEmail = e.Email
	}
	return outName, outEmail
}

// parseMailmap parses a .mailmap blob into a mailmap. Comments (#) and blank
// lines are skipped; malformed lines are silently dropped.
func parseMailmap(blob []byte) *mailmap {
	m := &mailmap{
		byEmail:        map[string]mailmapEntry{},
		byNameAndEmail: map[string]mailmapEntry{},
	}
	scan := bufio.NewScanner(bytes.NewReader(blob))
	for scan.Scan() {
		line := scan.Text()
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parsed, ok := parseMailmapLine(line)
		if !ok {
			continue
		}
		entry := mailmapEntry{Name: parsed.properName, Email: parsed.properEmail}
		commitEmail := strings.ToLower(parsed.commitEmail)
		if parsed.commitName != "" {
			m.byNameAndEmail[parsed.commitName+"\x00"+commitEmail] = entry
		} else {
			m.byEmail[commitEmail] = entry
		}
	}
	return m
}

type parsedMailmapLine struct {
	properName  string
	properEmail string
	commitName  string
	commitEmail string
}

// parseMailmapLine pulls the <…> email brackets out of a line in source
// order. The last email is the commit (lookup) email. If there are two
// emails, the first is the proper (replacement) email and any free text
// between the two is the commit (lookup) name. Free text before the first
// email is the proper (replacement) name.
func parseMailmapLine(line string) (parsedMailmapLine, bool) {
	var out parsedMailmapLine
	type bracket struct{ start, end int }
	var brs []bracket
	for i := 0; i < len(line); i++ {
		if line[i] != '<' {
			continue
		}
		for j := i + 1; j < len(line); j++ {
			if line[j] == '>' {
				brs = append(brs, bracket{i, j})
				i = j
				break
			}
		}
	}
	if len(brs) == 0 {
		return out, false
	}
	if len(brs) == 1 {
		out.commitEmail = strings.TrimSpace(line[brs[0].start+1 : brs[0].end])
		out.properName = strings.TrimSpace(line[:brs[0].start])
		return out, out.commitEmail != ""
	}
	first, last := brs[0], brs[len(brs)-1]
	out.properEmail = strings.TrimSpace(line[first.start+1 : first.end])
	out.commitEmail = strings.TrimSpace(line[last.start+1 : last.end])
	out.properName = strings.TrimSpace(line[:first.start])
	out.commitName = strings.TrimSpace(line[first.end+1 : last.start])
	return out, out.commitEmail != ""
}

// loadMailmapFromTree reads .mailmap from the given tree and parses it, or
// returns nil when the file is absent or unreadable.
func loadMailmapFromTree(tree *object.Tree) *mailmap {
	if tree == nil {
		return nil
	}
	f, err := tree.File(".mailmap")
	if err != nil {
		return nil
	}
	reader, err := f.Reader()
	if err != nil {
		return nil
	}
	defer reader.Close()
	blob, err := io.ReadAll(reader)
	if err != nil {
		return nil
	}
	return parseMailmap(blob)
}
