// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Used as quick lookup for files with the same name to avoid some processing
// needs to be sync.Map as it potentially could be called by many GoRoutines
var extensionCache sync.Map

// Added as a way to track files per run.
var visitedPaths sync.Map

// A custom version of extracting extensions for a file
// which also has a case-insensitive cache in order to save
// some needless processing
func getExtension(name string) string {
	name = strings.ToLower(name)
	extension, ok := extensionCache.Load(name)

	if ok {
		return extension.(string)
	}

	ext := filepath.Ext(name)

	if ext == "" || strings.LastIndex(name, ".") == 0 {
		extension = name
	} else {
		// Handling multiple dots or multiple extensions only needs to delete the last extension
		// and then call filepath.Ext.
		// If there are multiple extensions, it is the value of subExt,
		// otherwise subExt is an empty string.
		subExt := filepath.Ext(strings.TrimSuffix(name, ext))
		extension = strings.TrimPrefix(subExt+ext, ".")
	}

	extensionCache.Store(name, extension)
	return extension.(string)
}

func cleanVisitedPaths() {
	visitedPaths.Clear()
}

// nameGuess is what can be worked out about a file from its path and name, with
// no system call. detectForJob does that work and newFileJob finishes the job
// with what a stat says.
type nameGuess struct {
	language    []string
	extension   string
	unsupported bool
}

// detectForJob decides from the path and the name alone whether a file is one
// scc will count, and what it thinks the file is. It reports false where the
// name settles it, which is every rule below: the extension is unknown, or it
// is not on the allow list, or it is on an exclude list.
//
// It is split out so the caller can ask before it stats. Every one of these
// rules used to be applied after the stat, so a run told to count one language
// still paid a stat for every file in the tree: counting the C of the Linux
// kernel stat'd 91,158 files to keep 54,000 of them.
func detectForJob(path, name string) (nameGuess, bool) {
	language, extension := DetectLanguage(name)

	// Path pattern count rules can relabel a file to a new minted category and
	// can also rescue files that normal detection would otherwise skip. First
	// matching rule wins, evaluated against the full path as supplied.
	if len(compiledCountRules) != 0 {
		for _, r := range compiledCountRules {
			if r.re.MatchString(path) {
				language = []string{r.name}
				LoadLanguageFeature(r.name)
				break
			}
		}
	}

	// If detection found nothing we normally skip the file. With
	// --count-unsupported we instead count it under the "Unknown" category as
	// plain text (issue #464). Binary files are still dropped later once a null
	// byte is seen while counting, so this only surfaces text files scc doesn't
	// yet recognise.
	unsupported := false
	if len(language) == 0 {
		if !CountUnsupported {
			printWarnF("skipping file unknown extension: %s", name)
			return nameGuess{}, false
		}
		language = []string{UnknownLanguage}
		unsupported = true
	}

	// check if extensions in the allow list, which should limit to just those extensions
	if len(AllowListExtensions) != 0 {
		ok := false
		for _, x := range AllowListExtensions {
			if x == extension {
				ok = true
			}
		}

		if !ok {
			printWarnF("skipping file as not in allow list: %s", name)
			return nameGuess{}, false
		}
	}

	// check if we should exclude this type
	if len(ExcludeListExtensions) != 0 {
		ok := true
		for _, x := range ExcludeListExtensions {
			if x == extension {
				ok = false
			}
		}

		if !ok {
			printWarnF("skipping file as in exclude list: %s", name)
			return nameGuess{}, false
		}
	}

	if len(ExcludeFilename) != 0 {
		ok := true
		for _, x := range ExcludeFilename {
			if strings.Contains(name, x) {
				ok = false
			}
		}

		if !ok {
			printWarnF("skipping file as in exclude file list: %s", name)
			return nameGuess{}, false
		}
	}

	return nameGuess{language: language, extension: extension, unsupported: unsupported}, true
}

// newFileJob works out what a file is from its name and then finishes it with
// what the stat says. A caller that has already asked detectForJob calls
// newFileJobGuessed instead rather than ask twice.
func newFileJob(path, name string, fileInfo os.FileInfo) *FileJob {
	guess, ok := detectForJob(path, name)
	if !ok {
		return nil
	}

	return newFileJobGuessed(path, name, fileInfo, guess)
}

func newFileJobGuessed(path, name string, fileInfo os.FileInfo, guess nameGuess) *FileJob {
	if NoLarge {
		if fileInfo.Size() >= LargeByteCount {
			printWarnF("skipping large file due to byte size: %s", path)
			return nil
		}
	}

	var symPath = ""
	// Check if the file is a symlink and if we want to count those then work out its path and rejig
	// everything so we can count the real file to ensure the counts are correct
	if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
		if !IncludeSymLinks {
			printWarnF("skipping symlink file: %s", name)
			return nil
		}

		var err error
		symPath, err = filepath.EvalSymlinks(path)
		if err != nil {
			printError(err.Error())
			return nil
		}
		fileInfo, err = os.Lstat(symPath)
		if err != nil {
			printError(err.Error())
			return nil
		}
	}

	// Skip non-regular files. They are unlikely to be code and may hang if we
	// try and read them.
	if !fileInfo.Mode().IsRegular() {
		printWarnF("skipping non-regular file: %s", path)
		return nil
	}

	// This determines the real path
	realPath := path
	if symPath != "" {
		realPath = symPath
	}

	// Prevent duplicate processing and loops
	if _, exists := visitedPaths.LoadOrStore(realPath, true); exists {
		printWarnF("skipping already processed file: %s", realPath)
		return nil
	}

	// Unknown carries no language features by design so there is nothing to
	// load; loading it would also miss the language database entirely.
	if !guess.unsupported {
		for _, l := range guess.language {
			LoadLanguageFeature(l)
		}
	}

	if !CountIgnore {
		for _, l := range guess.language {
			if l == "ignore" || l == "gitignore" {
				return nil
			}
		}
	}

	return &FileJob{
		Location:          path,
		Symlocation:       symPath,
		Filename:          name,
		Extension:         guess.extension,
		PossibleLanguages: guess.language,
		Bytes:             fileInfo.Size(),
	}
}
