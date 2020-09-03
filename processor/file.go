package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/boyter/scc/processor/gitignore"
	"github.com/dbaggerman/cuba"
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

// DirectoryJob is a struct for dealing with directories we want to walk
type DirectoryJob struct {
	root    string
	path    string
	ignores []gitignore.IgnoreMatcher
}

// DirectoryWalker is responsible for actually walking directories using cuba
type DirectoryWalker struct {
	buffer   *cuba.Pool
	output   chan<- *FileJob
	excludes []*regexp.Regexp
}

// NewDirectoryWalker create the new directory walker
func NewDirectoryWalker(output chan<- *FileJob) *DirectoryWalker {
	directoryWalker := &DirectoryWalker{
		output: output,
	}
	for _, exclude := range Exclude {
		directoryWalker.excludes = append(directoryWalker.excludes, regexp.MustCompile(exclude))
	}

	directoryWalker.buffer = cuba.New(directoryWalker.Walk, cuba.NewStack())
	directoryWalker.buffer.SetMaxWorkers(int32(DirectoryWalkerJobWorkers))

	return directoryWalker
}

// Start actually starts directory traversal
func (dw *DirectoryWalker) Start(root string) error {
	root = filepath.Clean(root)

	fileInfo, err := os.Lstat(root)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		fileJob := newFileJob(root, filepath.Base(root), fileInfo)
		if fileJob != nil {
			dw.output <- fileJob
		}

		return nil
	}

	_ = dw.buffer.Push(
		&DirectoryJob{
			root:    root,
			path:    root,
			ignores: nil,
		},
	)

	return nil
}

// Run continues to run everything
func (dw *DirectoryWalker) Run() {
	dw.buffer.Finish()
	close(dw.output)
}

// Walk walks the directory as quickly as it can
func (dw *DirectoryWalker) Walk(handle *cuba.Handle) {
	job := handle.Item().(*DirectoryJob)

	ignores := job.ignores

	dirents, err := dw.Readdir(job.path)
	if err != nil {
		printError(err.Error())
		return
	}

	for _, dirent := range dirents {
		name := dirent.Name()

		if (!GitIgnore && name == ".gitignore") || (!Ignore && name == ".ignore") {
			path := filepath.Join(job.path, name)

			ignore, err := gitignore.NewGitIgnore(path)
			if err != nil {
				printError(fmt.Sprintf("failed to load gitignore %s: %v", job.path, err))
			} else {
				ignores = append(ignores, ignore)
			}
		}
	}

DIRENTS:
	for _, dirent := range dirents {
		name := dirent.Name()
		path := filepath.Join(job.path, name)
		isDir := dirent.IsDir()

		for _, deny := range PathDenyList {
			if strings.HasSuffix(path, deny) {
				if Verbose {
					printWarn(fmt.Sprintf("skipping directory due to being in denylist: %s", path))
				}
				continue DIRENTS
			}
		}

		for _, exclude := range dw.excludes {
			if exclude.Match([]byte(name)) || exclude.Match([]byte(path)) {
				if Verbose {
					printWarn("skipping file/directory due to match exclude: " + name)
				}
				continue DIRENTS
			}
		}

		for _, ignore := range ignores {
			if ignore.Match(path, isDir) {
				if Verbose {
					printWarn("skipping directory due to ignore: " + path)
				}
				continue DIRENTS
			}
		}

		if isDir {
			handle.Push(
				&DirectoryJob{
					root:    job.root,
					path:    path,
					ignores: ignores,
				},
			)
		} else {
			fileJob := newFileJob(path, name, dirent)
			if fileJob != nil {
				dw.output <- fileJob
			}
		}
	}
}

// Readdir reads a directory such that we know what files are in there
func (dw *DirectoryWalker) Readdir(path string) ([]os.FileInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return []os.FileInfo{}, fmt.Errorf("failed to open %s: %v", path, err)
	}
	defer file.Close()

	dirents, err := file.Readdir(-1)
	if err != nil {
		return []os.FileInfo{}, fmt.Errorf("failed to read %s: %v", path, err)
	}

	return dirents, nil
}

func newFileJob(path, name string, fileInfo os.FileInfo) *FileJob {
	if NoLarge {
		if fileInfo.Size() >= LargeByteCount {
			if Verbose {
				printWarn(fmt.Sprintf("skipping large file due to byte size: %s", path))
			}
			return nil
		}
	}

	var symPath = ""
	// Check if the file is a symlink and if we want to count those then work out its path and rejig
	// everything so we can count the real file to ensure the counts are correct
	if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
		if !IncludeSymLinks {
			if Verbose {
				printWarn(fmt.Sprintf("skipping symlink file: %s", name))
			}

			return nil
		}

		symPath, _ = filepath.EvalSymlinks(path)
		fileInfo, _ = os.Lstat(symPath)
	}

	language, extension := DetectLanguage(name)

	if len(language) != 0 {
		if len(AllowListExtensions) != 0 {
			ok := false
			for _, x := range AllowListExtensions {
				if x == extension {
					ok = true
				}
			}

			if !ok {
				if Verbose {
					printWarn(fmt.Sprintf("skipping file as not in allow list: %s", name))
				}
				return nil
			}
		}

		for _, l := range language {
			LoadLanguageFeature(l)
		}

		return &FileJob{
			Location:          path,
			Symlocation:       symPath,
			Filename:          name,
			Extension:         extension,
			PossibleLanguages: language,
			Bytes:             fileInfo.Size(),
		}
	} else if Verbose {
		printWarn(fmt.Sprintf("skipping file unknown extension: %s", name))
	}

	return nil
}
