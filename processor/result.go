// SPDX-License-Identifier: MIT

package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/boyter/gocodewalker"
)

// ProcessResult runs the same pipeline as Process but returns structured results
// instead of formatting to stdout. Useful for programmatic consumers like MCP servers.
func ProcessResult() ([]LanguageSummary, error) {
	ProcessConstants()
	processFlags()
	cleanVisitedPaths()

	if len(DirFilePaths) == 0 {
		DirFilePaths = append(DirFilePaths, ".")
	}

	filePaths := []string{}
	dirPaths := []string{}

	for _, f := range DirFilePaths {
		fpath := filepath.Clean(f)

		s, err := os.Stat(fpath)
		if err != nil {
			return nil, fmt.Errorf("file or directory could not be read: %s", fpath)
		}

		if s.IsDir() {
			dirPaths = append(dirPaths, fpath)
		} else {
			filePaths = append(filePaths, fpath)
		}
	}

	SortBy = strings.ToLower(SortBy)
	ctx := processorContext{remap: newRemapConfig(RemapAll, RemapUnknown)}

	printDebugF("NumCPU: %d", runtime.NumCPU())
	printDebugF("SortBy: %s", SortBy)
	printDebugF("PathDenyList: %v", PathDenyList)

	potentialFilesQueue := make(chan *gocodewalker.File, FileListQueueSize)
	fileListQueue := make(chan *FileJob, FileListQueueSize)
	fileSummaryJobQueue := make(chan *FileJob, FileSummaryJobQueueSize)

	fileWalker := gocodewalker.NewParallelFileWalker(dirPaths, potentialFilesQueue)
	fileWalker.SetErrorHandler(func(e error) bool {
		printError(e.Error())
		return true
	})
	fileWalker.IgnoreGitIgnore = GitIgnore
	fileWalker.IgnoreIgnoreFile = Ignore
	fileWalker.IgnoreGitModules = GitModuleIgnore
	fileWalker.IncludeHidden = true
	fileWalker.ExcludeDirectory = PathDenyList
	fileWalker.SetConcurrency(DirectoryWalkerJobWorkers)

	if !SccIgnore {
		fileWalker.CustomIgnore = []string{".sccignore"}
	}

	var excludePathRegexes []*regexp.Regexp
	for _, exclude := range Exclude {
		regexpResult, err := regexp.Compile(exclude)
		if err == nil {
			fileWalker.ExcludeFilenameRegex = append(fileWalker.ExcludeFilenameRegex, regexpResult)
			fileWalker.ExcludeDirectoryRegex = append(fileWalker.ExcludeDirectoryRegex, regexpResult)
			excludePathRegexes = append(excludePathRegexes, regexpResult)
		} else {
			printError(err.Error())
		}
	}

	go func() {
		err := fileWalker.Start()
		if err != nil {
			printError(err.Error())
		}
	}()

	go func() {
		for _, f := range filePaths {
			fileInfo, err := os.Lstat(f)
			if err != nil {
				continue
			}

			fileJob := newFileJob(f, f, fileInfo)
			if fileJob != nil {
				fileListQueue <- fileJob
			}
		}

		for fi := range potentialFilesQueue {
			shouldExclude := false
			for _, re := range excludePathRegexes {
				if re.MatchString(fi.Location) {
					shouldExclude = true
					break
				}
			}
			if shouldExclude {
				continue
			}

			fileInfo, err := os.Lstat(fi.Location)
			if err != nil {
				continue
			}

			if !fileInfo.IsDir() {
				fileJob := newFileJob(fi.Location, fi.Filename, fileInfo)
				if fileJob != nil {
					fileListQueue <- fileJob
				}
			}
		}
		close(fileListQueue)
	}()

	go ctx.fileProcessorWorker(fileListQueue, fileSummaryJobQueue)

	language := aggregateLanguageSummary(fileSummaryJobQueue)
	language = sortLanguageSummary(language)

	return language, nil
}
