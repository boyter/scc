package processor

import (
	"fmt"
	"github.com/karrick/godirwalk"
	"github.com/monochromegane/go-gitignore"
	"io/ioutil"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"github.com/itsmontoya/mailbox"
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

	locs := strings.Split(name, ".")

	switch {
	case len(locs) == 0 || len(locs) == 1 || strings.LastIndex(name, ".") == 0:
		extension = name
	case len(locs) == 2:
		extension = locs[len(locs)-1]
	default:
		extension = locs[len(locs)-2] + "." + locs[len(locs)-1]
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
func walkDirectoryParallel(root string, output *mailbox.Mailbox) {
	startTime := makeTimestampMilli()
	blackList := strings.Split(PathBlacklist, ",")
	whiteList := strings.Split(WhiteListExtensions, ",")
	extensionLookup := ExtensionToLanguage

	// If input has a supplied white list of extensions then loop through them
	// and modify the lookup we use to cut down on extra checks
	if len(WhiteListExtensions) != 0 {
		wlExtensionLookup := map[string]string{}

		for _, white := range whiteList {
			language, ok := extensionLookup[white]

			if ok {
				wlExtensionLookup[white] = language
			}
		}

		extensionLookup = wlExtensionLookup
	}

	var mutex = &sync.Mutex{}
	totalCount := 0

	var wg sync.WaitGroup
	all, _ := ioutil.ReadDir(root)
	// TODO the gitignore should check for futher gitignores deeper in the tree
	gitignore, gitignoreerror := gitignore.NewGitIgnore(filepath.Join(root, ".gitignore"))

	for _, f := range all {
		// Godirwalk despite being faster than the default walk is still too slow to feed the
		// CPU's and so we need to walk in parallel to keep up as much as possible
		if f.IsDir() {
			// Need to check if the directory is in the blacklist and if so don't bother adding a goroutine to process it
			shouldSkip := false
			for _, black := range blackList {
				if strings.HasPrefix(filepath.Join(root, f.Name()), black) {
					shouldSkip = true
					printWarn(fmt.Sprintf("skipping directory due to being in blacklist: %s", filepath.Join(root, f.Name())))
					break
				}
			}

			if !shouldSkip {
				wg.Add(1)
				go func(toWalk string) {
					filejobs := walkDirectory(toWalk, blackList, extensionLookup)
					for i := 0; i < len(filejobs); i++ {
						//*output <- &filejobs[i]
						//output.Put(&filejobs[i])
						output.Send(&filejobs[i], true)
					}

					mutex.Lock()
					totalCount += len(filejobs)
					mutex.Unlock()

					// Turn GC back to what it was before if we have parsed enough files
					if totalCount >= GcFileCount {
						debug.SetGCPercent(gcPercent)
					}
					wg.Done()
				}(filepath.Join(root, f.Name()))
			}
		} else {
			if gitignoreerror != nil || !gitignore.Match(filepath.Join(root, f.Name()), false) {
				extension := getExtension(f.Name())
				language, ok := extensionLookup[extension]

				if ok {
					//*output <- &FileJob{Location: filepath.Join(root, f.Name()), Filename: f.Name(), Extension: extension, Language: language}
					output.Send(&FileJob{Location: filepath.Join(root, f.Name()), Filename: f.Name(), Extension: extension, Language: language}, true)
					mutex.Lock()
					totalCount++
					mutex.Unlock()
				} else if Verbose {
					printWarn(fmt.Sprintf("skipping file unknown extension: %s", f.Name()))
				}
			}
		}
	}

	wg.Wait()
	//close(*output)
	output.Close()
	if Debug {
		printDebug(fmt.Sprintf("milliseconds to walk directory: %d", makeTimestampMilli()-startTime))
	}
}

func walkDirectory(toWalk string, blackList []string, extensionLookup map[string]string) []FileJob {
	extension := ""
	filejobs := []FileJob{}

	godirwalk.Walk(toWalk, &godirwalk.Options{
		// Unsorted is meant to make the walk faster and we need to sort after processing anyway
		Unsorted: true,
		Callback: func(root string, info *godirwalk.Dirent) error {
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
					filejobs = append(filejobs, FileJob{Location: root, Filename: info.Name(), Extension: extension, Language: language})
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
