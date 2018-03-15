package processor

import (
	"fmt"
	"github.com/karrick/godirwalk"
	"github.com/monochromegane/go-gitignore"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
)

var extensionCache sync.Map

func getExtension(name string) string {
	name = strings.ToLower(name)
	extension, ok := extensionCache.Load(name)

	if ok {
		return extension.(string)
	}

	loc := strings.LastIndex(name, ".")

	if loc != -1 {
		extension = name[loc+1:]
	} else {
		extension = name
	}

	extensionCache.Store(name, extension)
	return extension.(string)
}

// Get all the files that exist in the directory
func walkDirectory(root string, output *chan *FileJob) {
	startTime := makeTimestampMilli()

	var wg sync.WaitGroup
	all, _ := ioutil.ReadDir(root)
	gitignore, gitignoreerror := gitignore.NewGitIgnore(filepath.Join(root, ".gitignore"))

	for _, f := range all {
		if f.IsDir() {
			wg.Add(1)
			go func(toWalk string) {
				godirwalk.Walk(toWalk, &godirwalk.Options{
					Unsorted: true,
					Callback: func(root string, info *godirwalk.Dirent) error {
						// TODO this should be configurable via command line
						if strings.HasPrefix(root, ".git/") || strings.HasPrefix(root, ".hg/") || strings.HasPrefix(root, ".svn/") {
							printWarn(fmt.Sprintf("skipping directory due to being in blacklist: %s", root))
							return filepath.SkipDir
						}

						if !info.IsDir() {
							if gitignoreerror != nil || !gitignore.Match(filepath.Join(root, info.Name()), false) {

								extension := getExtension(info.Name())
								language, ok := ExtensionToLanguage[extension]

								if ok {
									*output <- &FileJob{Location: root, Filename: info.Name(), Extension: extension, Language: language}
								} else {
									printWarn(fmt.Sprintf("skipping file unknown extension: %s", info.Name()))
								}
							}
						}

						return nil
					},
					ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
						printDebug(fmt.Sprintf("error walking: %s %s", osPathname, err))
						return godirwalk.SkipNode
					},
				})
				wg.Done()
			}(filepath.Join(root, f.Name()))
		} else {
			if gitignoreerror != nil || !gitignore.Match(filepath.Join(root, f.Name()), false) {

				extension := getExtension(f.Name())
				language, ok := ExtensionToLanguage[extension]

				if ok {
					*output <- &FileJob{Location: root, Filename: f.Name(), Extension: extension, Language: language}
				} else {
					printWarn(fmt.Sprintf("skipping file unknown extension: %s", f.Name()))
				}
			}
		}
	}

	wg.Wait()

	close(*output)
	printDebug(fmt.Sprintf("milliseconds to walk directory: %d", makeTimestampMilli()-startTime))
}
