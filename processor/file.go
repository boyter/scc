package processor

import (
	"fmt"
	"github.com/karrick/godirwalk"
	"github.com/monochromegane/go-gitignore"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
)

// Used as quick lookup for files with the same name to avoid some processing
// needs to be sync.Map as it potentially could be called by many GoRoutines
var extensionCache sync.Map

// A custom version of extracting extensions for a file
// which also has a case insensitive cache in order to save
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

// Iterate over the supplied directory in parallel and each file that is not
// excluded by the .gitignore and we know the extension of add to the supplied
// channel. This attempts to span out in parallel based on the number of directories
// in the supplied directory. Tests using a single process showed no lack of performance
// even when hitting older spinning platter disks for this way
//func walkDirectoryParallel(root string, output *RingBuffer) {
func walkDirectoryParallel(root string, output chan *FileJob) {
	startTime := makeTimestampMilli()
	extensionLookup := ExtensionToLanguage

	// If input has a supplied white list of extensions then loop through them
	// and modify the lookup we use to cut down on extra checks
	if len(WhiteListExtensions) != 0 {
		wlExtensionLookup := map[string][]string{}

		for _, white := range WhiteListExtensions {
			language, ok := extensionLookup[white]

			if ok {
				wlExtensionLookup[white] = language
			}
		}

		extensionLookup = wlExtensionLookup
	}

	var totalCount int64 = 0

	var wg sync.WaitGroup

	var isSoloFile bool = false
	var all []os.FileInfo
	// clean path including trailing slashes
	root = filepath.Clean(root)
	target, _ := os.Lstat(root)
	if !target.IsDir() {
		// create an array with a single FileInfo
		all = append(all, target)
		isSoloFile = true
	} else {
		all, _ = ioutil.ReadDir(root)
	}

	// TODO the gitignore should check for further gitignores deeper in the tree
	gitignore, gitignoreerror := gitignore.NewGitIgnore(filepath.Join(root, ".gitignore"))
	resetGc := false

	var regex *regexp.Regexp

	if Exclude != "" {
		regex = regexp.MustCompile(Exclude)
	}

	var fpath string
	for _, f := range all {
		// Godirwalk despite being faster than the default walk is still too slow to feed the
		// CPU's and so we need to walk in parallel to keep up as much as possible
		if f.IsDir() {
			// Need to check if the directory is in the blacklist and if so don't bother adding a goroutine to process it
			shouldSkip := false
			for _, black := range PathBlacklist {
				if strings.HasPrefix(filepath.Join(root, f.Name()), black) {
					shouldSkip = true
					if Verbose {
						printWarn(fmt.Sprintf("skipping directory due to being in blacklist: %s", filepath.Join(root, f.Name())))
					}
					break
				}
			}

			if Exclude != "" {
				if regex.Match([]byte(f.Name())) {
					if Verbose {
						printWarn("skipping directory due to match exclude: " + f.Name())
					}
					shouldSkip = true
				}
			}

			if !shouldSkip {
				wg.Add(1)
				go func(toWalk string) {
					filejobs := walkDirectory(toWalk, PathBlacklist, extensionLookup)
					for i := 0; i < len(filejobs); i++ {
						for _, lan := range filejobs[i].PossibleLanguages {
							LoadLanguageFeature(lan)
						}
						output <- &filejobs[i]
					}

					atomic.AddInt64(&totalCount, int64(len(filejobs)))

					// Turn GC back to what it was before if we have parsed enough files
					if !resetGc && atomic.LoadInt64(&totalCount) >= int64(GcFileCount) {
						debug.SetGCPercent(gcPercent)
						resetGc = true
					}
					wg.Done()
				}(filepath.Join(root, f.Name()))
			}
		} else { // File processing starts here
			if isSoloFile {
				fpath = root
			} else {
				fpath = filepath.Join(root, f.Name())
			}
			if gitignoreerror != nil || !gitignore.Match(fpath, false) {

				shouldSkip := false
				if Exclude != "" {
					if regex.Match([]byte(f.Name())) {
						if Verbose {
							printWarn("skipping file due to match exclude: " + f.Name())
						}
						shouldSkip = true
					}
				}

				if !shouldSkip {
					extension := ""
					// Lookup in case the full name matches
					language, ok := extensionLookup[strings.ToLower(f.Name())]

					// If no match check if we have a matching extension
					if !ok {
						extension = getExtension(f.Name())
						language, ok = extensionLookup[extension]
					}

					// Convert from d.ts to ts and check that in case of multiple extensions
					if !ok {
						language, ok = extensionLookup[getExtension(extension)]
					}

					if ok {
						atomic.AddInt64(&totalCount, 1)

						for _, l := range language {
							LoadLanguageFeature(l)
						}

						output <- &FileJob{Location: fpath, Filename: f.Name(), Extension: extension, PossibleLanguages: language}
					} else if Verbose {
						printWarn(fmt.Sprintf("skipping file unknown extension: %s", f.Name()))
					}
				}
			}
		}
	}

	wg.Wait()
	close(output)
	if Debug {
		printDebug(fmt.Sprintf("milliseconds to walk directory: %d", makeTimestampMilli()-startTime))
	}
}

func walkDirectory(toWalk string, blackList []string, extensionLookup map[string][]string) []FileJob {
	extension := ""
	var filejobs []FileJob

	var regex *regexp.Regexp
	if Exclude != "" {
		regex = regexp.MustCompile(Exclude)
	}

	_ = godirwalk.Walk(toWalk, &godirwalk.Options{
		// Unsorted is meant to make the walk faster and we need to sort after processing anyway
		Unsorted: true,
		Callback: func(root string, info *godirwalk.Dirent) error {

			if Exclude != "" {
				if regex.Match([]byte(info.Name())) {
					if Verbose {
						if info.IsDir() {
							printWarn("skipping directory due to match exclude: " + root)
						} else {
							printWarn("skipping file due to match exclude: " + root)
						}
					}
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			if info.IsDir() {
				for _, black := range blackList {
					if strings.HasPrefix(root, black+"/") || strings.HasPrefix(root, black) {
						if Verbose {
							printWarn(fmt.Sprintf("skipping directory due to being in blacklist: %s", root))
						}
						return filepath.SkipDir
					}
				}
			}

			if !info.IsDir() {
				// Lookup in case the full name matches
				language, ok := extensionLookup[strings.ToLower(info.Name())]

				// If no match check if we have a matching extension
				if !ok {
					extension = getExtension(info.Name())
					language, ok = extensionLookup[extension]
				}

				// Convert from d.ts to ts and check that in case of multiple extensions
				if !ok {
					language, ok = extensionLookup[getExtension(extension)]
				}

				if ok {
					filejobs = append(filejobs, FileJob{Location: root, Filename: info.Name(), Extension: extension, PossibleLanguages: language})
				} else if Verbose {
					printWarn(fmt.Sprintf("skipping file unknown extension: %s", info.Name()))
				}
			}

			return nil
		},
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			if Verbose {
				printWarn(fmt.Sprintf("error walking: %s %s", osPathname, err))
			}
			return godirwalk.SkipNode
		},
	})

	return filejobs
}
