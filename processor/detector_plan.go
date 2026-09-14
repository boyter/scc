// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

// The heuristic prefilter decides whether a heuristic's regex is worth running.
// Done literal by literal it is one full scan of the file per literal, and a
// header that is neither C++ nor Objective-C has to be proved to contain none
// of them: on the Linux kernel that is 37 scans of up to 20KB for each of
// 26,000 .h files, and it came to 12% of the whole run.
//
// A plan does the same job in a handful of passes. It is built once per set of
// candidate languages and then:
//
//   - every anchored literal is answered by one walk of the line starts, since
//     an anchored literal can only sit after a line's leading spaces and tabs;
//   - unanchored literals that share a first byte are answered together by one
//     scan for that byte, which turns the fifteen `#include <...>` literals and
//     the eight `@`-prefixed Objective-C literals into two passes;
//   - what is left keeps a scan each.
//
// The plan only ever decides whether to run a regex, so it is allowed to say
// "possible" too often. It must never say "impossible" when the old literal
// check would have said otherwise, and the tests check exactly that.

// planGroupMin is the number of literals that have to share a first byte before
// scanning for that byte and comparing beats a scan per literal. Below it the
// per-literal scan wins, because bytes.Index filters on more than one byte and
// a group of one or two pays a comparison at every occurrence of a byte that
// may well be common.
const planGroupMin = 3

// isAnchorSkip reports whether a byte is one the line walk steps over before
// looking for an anchored literal. It is the union of what both leads accept
// plus the vertical tab, which neither does, so it is only ever the first of
// two tests: scanLineStarts records what it actually stepped over and the slot
// gate below decides whether that was legal for the lead in hand.
//
// It was one test when the walk only proposed a candidate for the regex to
// confirm, where skipping more than a lead accepts cost a regex run and never
// an answer. The anchored form made the walk the thing that decides, and an
// over-skip became a match at a position the pattern could not have started
// at. Three headers of the corpus in TestAnchorSkipMatchesTheLead are what that
// looked like.
func isAnchorSkip(b byte) bool {
	return anchorSkip[b]
}

var anchorSkip = func() [256]bool {
	var t [256]bool
	for _, b := range []byte{' ', '\t', '\r', '\f', '\v'} {
		t[b] = true
	}
	return t
}()

type planHeuristic struct {
	re  *regexp.Regexp
	ids []int32 // literal ids, any of which lets the regex run; empty always runs
	// slot indexes the plan's anchored regexes when this heuristic is settled by
	// the line walk rather than by scanning the file with re, and is -1 when it
	// is not. See anchoredForm.
	slot int32
}

// anchoredLead is the leading "start of a line, then its indentation" that an
// anchored heuristic's pattern is written with. Everything after it can only
// match where the indentation ends, which is a place the line walk already
// stops at, so the pattern does not have to be searched for.
var anchoredLead = regexp.MustCompile(`^\(\?m\)\^(?:\\s\*|\[ \\t\]\*)`)

// anchoredForm rewrites an anchored heuristic's pattern into one that only ever
// matches at the start of the text it is given, so it can be tried at a known
// position instead of searched for across the file.
//
// The rewrite is worth the trouble because a pattern beginning ^\s* has no
// literal prefix, so regexp cannot skip ahead and instead tries every byte
// offset in the file: on LLVM's headers that is twenty times the work of trying
// the handful of offsets where the pattern could actually begin.
//
// It only fires on a pattern whose shape it recognises, and returns nil
// otherwise, so a hand written pattern that does something cleverer is left to
// the ordinary search.
// The second return says the lead was [ \t]*, which accepts a narrower run of
// indentation than \s* does and so gates the walk more tightly.
func anchoredForm(pattern string) (*regexp.Regexp, bool) {
	lead := anchoredLead.FindString(pattern)
	if lead == "" {
		return nil, false
	}
	tabOnly := strings.HasSuffix(lead, `[ \t]*`)

	rest := pattern[len(lead):]
	if rest == "" {
		return nil, false
	}

	// A pattern that begins by asking about the text before it means something
	// different when the text is cut at that point, so leave it alone. \A and \z
	// go the same way: the rewrite supplies its own \A.
	for _, bad := range []string{`\b`, `\B`, `\A`, `\z`, `\Z`} {
		if strings.HasPrefix(rest, bad) {
			return nil, false
		}
	}

	// (?m) is kept so a $ inside the pattern still means end of line, and \A
	// pins the whole thing to where it is tried.
	re, err := regexp.Compile(`(?m)\A(?:` + rest + `)`)
	if err != nil {
		return nil, false
	}

	return re, tabOnly
}

// anchoredLiterals reports whether a heuristic's literals can stand in for
// where its anchored pattern begins: each has to be something the line walk can
// find, which means non empty and not starting with the whitespace the walk
// steps over.
func anchoredLiterals(literals [][]byte) bool {
	if len(literals) == 0 {
		return false
	}
	for _, lit := range literals {
		if len(lit) == 0 || isAnchorSkip(lit[0]) {
			return false
		}
	}
	return true
}

type planLanguage struct {
	name       string
	heuristics []planHeuristic
}

// planFallback is a candidate language with no heuristics of its own, which is
// settled by counting its keywords instead. The keyword bytes are held here so
// the count does not have to go back to LanguageFeatures, and its mutex, for
// every file.
type planFallback struct {
	name     string
	keywords [][]byte
}

// planGroup is a set of literals sharing a first byte, answered by one scan for
// that byte. The fifteen C++ headers all begin '<', so at every '<' in a file
// the naive form compares all fifteen; second sorts them by their second byte so
// that "<linux/foo.h>" is compared against "<list>" and nothing else.
type planGroup struct {
	first  byte
	ids    []int32
	second [256][]int32 // ids keyed by the literal's second byte
	short  []int32      // one byte literals, which have no second byte
}

// planSingle is a literal scanned on its own, with the substring actually
// searched for. See pickScreen.
type planSingle struct {
	id     int32
	screen []byte
	off    int // where screen sits inside the literal
}

type heuristicPlan struct {
	langs   []planLanguage
	lits    [][]byte
	nlits   int
	anchor  [256][]int32 // anchored literals keyed by their first byte
	anchTop [256]bool    // whether anchor holds anything for a byte
	hasAnch bool
	groups  []planGroup  // unanchored literals sharing a first byte
	singles []planSingle // unanchored literals scanned one at a time

	// Heuristics settled by the line walk, in place of a search of the file.
	// anchRe is indexed by a heuristic's slot, and litSlots says which of them a
	// literal being found at a line start is worth trying.
	anchRe []*regexp.Regexp
	// anchTabOnly is indexed like anchRe and says the slot's lead was [ \t]*,
	// so the walk may only settle it where the indentation it stepped over was
	// spaces and tabs alone.
	anchTabOnly []bool
	litSlots    [][]int32

	// What the keyword count falls back to when no heuristic matched.
	fallbacks []planFallback
	primary   string
}

// planCacheEntry is one candidate set and the plan built for it.
type planCacheEntry struct {
	langs []string
	plan  *heuristicPlan
}

var (
	heuristicPlanMutex sync.Mutex
	heuristicPlans     atomic.Pointer[[]planCacheEntry]
)

// planFor returns the scan plan for a set of candidate languages, building it
// on first use. The candidate set comes from the extension lookup, so there are
// only ever a handful of them in a run, which is why the cache is a list read
// without a lock rather than a map: joining the names into a key allocated on
// every file counted, and the read lock around the map put an atomic write on a
// line every worker was touching.
func planFor(possibleLanguages []string) *heuristicPlan {
	if entries := heuristicPlans.Load(); entries != nil {
		for i := range *entries {
			if slices.Equal((*entries)[i].langs, possibleLanguages) {
				return (*entries)[i].plan
			}
		}
	}

	heuristicPlanMutex.Lock()
	defer heuristicPlanMutex.Unlock()

	// Another worker may have built this set while we waited.
	old := heuristicPlans.Load()
	if old != nil {
		for i := range *old {
			if slices.Equal((*old)[i].langs, possibleLanguages) {
				return (*old)[i].plan
			}
		}
	}

	plan := buildHeuristicPlan(possibleLanguages)

	var next []planCacheEntry
	if old != nil {
		next = make([]planCacheEntry, 0, len(*old)+1)
		next = append(next, *old...)
	}
	next = append(next, planCacheEntry{langs: slices.Clone(possibleLanguages), plan: plan})
	heuristicPlans.Store(&next)

	return plan
}

func buildHeuristicPlan(possibleLanguages []string) *heuristicPlan {
	plan := &heuristicPlan{}

	type litKey struct {
		s        string
		anchored bool
	}
	ids := map[litKey]int32{}
	anchored := map[int32]bool{}
	litSlots := map[int32][]int32{}

	intern := func(lit []byte, isAnchored bool) int32 {
		k := litKey{s: string(lit), anchored: isAnchored}
		if id, ok := ids[k]; ok {
			return id
		}
		id := int32(len(plan.lits))
		ids[k] = id
		plan.lits = append(plan.lits, lit)
		anchored[id] = isAnchored
		return id
	}

	for _, lan := range possibleLanguages {
		LanguageFeaturesMutex.Lock()
		langFeatures := LanguageFeatures[lan]
		LanguageFeaturesMutex.Unlock()

		if len(langFeatures.Heuristics) == 0 {
			// No heuristics, so this one is settled by counting keywords, and
			// one with no keywords either is the primary: the language a file
			// falls back to when nothing else is convincing. When more than one
			// qualifies the alphabetically first is taken so the choice is
			// deterministic.
			plan.fallbacks = append(plan.fallbacks, planFallback{name: lan, keywords: langFeatures.KeywordBytes})
			if len(langFeatures.Keywords) == 0 && (plan.primary == "" || lan < plan.primary) {
				plan.primary = lan
			}
			continue
		}

		pl := planLanguage{name: lan}
		for _, h := range langFeatures.Heuristics {
			ph := planHeuristic{re: h.Re, slot: -1}

			// An anchored heuristic whose pattern can be pinned is answered by
			// the line walk, which tries it only where its literals sit rather
			// than searching the file for it.
			if h.Anchored && anchoredLiterals(h.Literals) {
				if anch, tabOnly := anchoredForm(h.Re.String()); anch != nil {
					ph.slot = int32(len(plan.anchRe))
					plan.anchRe = append(plan.anchRe, anch)
					plan.anchTabOnly = append(plan.anchTabOnly, tabOnly)
					for _, lit := range h.Literals {
						id := intern(lit, true)
						litSlots[id] = append(litSlots[id], ph.slot)
						// The ids are kept as well, so the heuristic still
						// has the ordinary pre-check behind it and the plan's
						// own property test still covers these literals.
						ph.ids = append(ph.ids, id)
					}
					pl.heuristics = append(pl.heuristics, ph)
					continue
				}
			}

			for _, lit := range h.Literals {
				if len(lit) == 0 {
					// A zero length literal matches everywhere, so the regex
					// has to run regardless. Drop the ids so it always does.
					ph.ids = nil
					break
				}
				// An anchored literal that starts with the whitespace the
				// anchor skips cannot be answered by the line walk, so treat
				// the heuristic as always worth running rather than get it
				// wrong.
				if h.Anchored && isAnchorSkip(lit[0]) {
					ph.ids = nil
					break
				}
				ph.ids = append(ph.ids, intern(lit, h.Anchored))
			}
			pl.heuristics = append(pl.heuristics, ph)
		}
		plan.langs = append(plan.langs, pl)
	}

	plan.nlits = len(plan.lits)

	if len(litSlots) != 0 {
		plan.litSlots = make([][]int32, plan.nlits)
		for id, slots := range litSlots {
			plan.litSlots[id] = slots
		}
	}

	// Split the literals into the line walk, the shared first byte scans and
	// the leftovers.
	byFirst := map[byte][]int32{}
	for id, lit := range plan.lits {
		if anchored[int32(id)] {
			plan.anchor[lit[0]] = append(plan.anchor[lit[0]], int32(id))
			plan.anchTop[lit[0]] = true
			plan.hasAnch = true
			continue
		}
		byFirst[lit[0]] = append(byFirst[lit[0]], int32(id))
	}

	// Iterate the first bytes in order so a plan is the same however the map
	// happened to be laid out.
	for b := 0; b < 256; b++ {
		group, ok := byFirst[byte(b)]
		if !ok {
			continue
		}
		if len(group) >= planGroupMin {
			g := planGroup{first: byte(b), ids: group}
			for _, id := range group {
				if lit := plan.lits[id]; len(lit) == 1 {
					g.short = append(g.short, id)
				} else {
					g.second[lit[1]] = append(g.second[lit[1]], id)
				}
			}
			plan.groups = append(plan.groups, g)
			continue
		}
		for _, id := range group {
			off, screen := pickScreen(plan.lits[id])
			plan.singles = append(plan.singles, planSingle{id: id, screen: screen, off: off})
		}
	}

	return plan
}

// byteCost is roughly how often a byte turns up in source, in parts per
// thousand. It is only ever used to choose which part of a literal to scan for,
// so a value being off costs a little speed and never an answer. The shape is
// the one every language shares: space and the newline dominate, then the
// letters in English frequency order because identifiers are English words,
// with the underscore up among them because snake_case is everywhere, then the
// digits, then punctuation, and almost nothing above ASCII.
var byteCost = func() [256]uint16 {
	var c [256]uint16
	for i := range c {
		c[i] = 1 // anything not named below, high bytes included
	}
	for i := 'A'; i <= 'Z'; i++ {
		c[i] = 4
	}
	for i := '0'; i <= '9'; i++ {
		c[i] = 6
	}
	for b, f := range map[byte]uint16{
		' ': 140, '\n': 31, '\t': 28, '\r': 2,
		'e': 53, 't': 36, 'a': 33, 'o': 32, 'i': 33, 'n': 31, 's': 32, 'r': 30,
		'h': 20, 'l': 22, 'd': 24, 'c': 21, 'u': 18, 'm': 15, 'f': 14, 'p': 14,
		'g': 11, 'w': 8, 'y': 8, 'b': 10, 'v': 7, 'k': 5, 'x': 4, 'j': 2, 'q': 2, 'z': 2,
		'_': 50,
	} {
		c[b] = f
	}
	return c
}()

// pickScreen chooses the substring of a literal that is cheapest to search for.
// bytes.Index walks the haystack looking for the needle's first byte, so a
// literal that starts with a byte as common as the underscore pays a restart
// every twentieth byte of a C header: scanning for "plusplus" instead of
// "__cplusplus" is twice the throughput for the same answer.
//
// The screen is a substring of the literal, so the literal cannot be present
// without it. A file the screen does not match cannot hold the literal, and one
// it does match is checked for the literal itself, which keeps the result
// exactly what a plain search would have given.
func pickScreen(lit []byte) (int, []byte) {
	best := 0
	for i, b := range lit {
		if byteCost[b] < byteCost[lit[best]] {
			best = i
		}
	}
	if best == 0 {
		return 0, nil
	}
	return best, lit[best:]
}

// present fills found with which of the plan's literals the content holds, and
// hit with which of the plan's anchored heuristics matched.
func (plan *heuristicPlan) present(content []byte, found, hit []bool) {
	if plan.hasAnch {
		plan.scanLineStarts(content, found, hit)
	}

	for i := range plan.groups {
		plan.scanGroup(content, found, &plan.groups[i])
	}

	for _, s := range plan.singles {
		if s.screen == nil {
			if bytes.Contains(content, plan.lits[s.id]) {
				found[s.id] = true
			}
			continue
		}

		// The literal can only sit at a fixed offset back from where its screen
		// matched, so each match is settled by a comparison there rather than by
		// a second search of the file. Walking the matches costs one pass for
		// the screen however many of them there are, which matters: "bute" is
		// the cheap way to look for __has_cpp_attribute but it also matches
		// every attribute and distribute in the kernel.
		lit := plan.lits[s.id]
		pos := 0
		for {
			i := bytes.Index(content[pos:], s.screen)
			if i < 0 {
				break
			}
			at := pos + i
			if start := at - s.off; start >= 0 && bytes.HasPrefix(content[start:], lit) {
				found[s.id] = true
				break
			}
			pos = at + 1
		}
	}
}

// scanLineStarts answers every anchored literal in one walk. An anchored
// literal has to sit at the start of a line preceded only by spaces and tabs,
// which is to say immediately after the line's indentation, so there is exactly
// one place per line worth looking at.
func (plan *heuristicPlan) scanLineStarts(content []byte, found, hit []bool) {
	pos := 0
	for pos < len(content) {
		j := pos
		// What was stepped over decides which slots may be settled here. [ \t]*
		// accepts spaces and tabs, \s* accepts those plus a carriage return and
		// a form feed, and neither accepts a vertical tab, which Go's \s leaves
		// out.
		tabOnly, goSpace := true, true
		for j < len(content) && isAnchorSkip(content[j]) {
			switch content[j] {
			case ' ', '\t':
			case '\v':
				tabOnly, goSpace = false, false
			default:
				tabOnly = false
			}
			j++
		}
		if j >= len(content) {
			return
		}

		// anchTop answers the common case, a line starting with a byte no
		// literal does, out of a byte of a 256 byte table rather than a slice
		// header out of a six kilobyte one.
		if plan.anchTop[content[j]] {
			for _, id := range plan.anchor[content[j]] {
				// A literal already found still has to be compared while a
				// heuristic of its own is undecided, because the pattern is
				// tried where the literal sits.
				slots := plan.slotsFor(id, hit)
				if found[id] && slots == nil {
					continue
				}
				if !bytes.HasPrefix(content[j:], plan.lits[id]) {
					continue
				}
				found[id] = true
				for _, s := range slots {
					if plan.anchTabOnly[s] {
						if !tabOnly {
							continue
						}
					} else if !goSpace {
						continue
					}
					if !hit[s] && plan.anchRe[s].Match(content[j:]) {
						hit[s] = true
					}
				}
			}
		}

		// The indentation is behind us, so the search for the end of the line
		// starts there rather than walking it a second time.
		n := bytes.IndexByte(content[j:], '\n')
		if n < 0 {
			return
		}
		pos = j + n + 1
	}
}

// slotsFor returns the anchored heuristics a literal is worth trying for, and
// nil once every one of them has already matched.
func (plan *heuristicPlan) slotsFor(id int32, hit []bool) []int32 {
	if plan.litSlots == nil {
		return nil
	}
	slots := plan.litSlots[id]
	for _, s := range slots {
		if !hit[s] {
			return slots
		}
	}
	return nil
}

// scanGroup answers a set of literals that share a first byte with one scan for
// that byte, comparing the ones not yet found at each occurrence.
func (plan *heuristicPlan) scanGroup(content []byte, found []bool, g *planGroup) {
	remaining := 0
	for _, id := range g.ids {
		if !found[id] {
			remaining++
		}
	}
	if remaining == 0 {
		return
	}

	pos := 0
	for {
		n := bytes.IndexByte(content[pos:], g.first)
		if n < 0 {
			return
		}
		pos += n

		rest := content[pos:]
		if len(rest) > 1 {
			for _, id := range g.second[rest[1]] {
				if found[id] {
					continue
				}
				if bytes.HasPrefix(rest, plan.lits[id]) {
					found[id] = true
					remaining--
				}
			}
		}
		for _, id := range g.short {
			if !found[id] {
				found[id] = true
				remaining--
			}
		}
		if remaining == 0 {
			return
		}

		pos++
	}
}
