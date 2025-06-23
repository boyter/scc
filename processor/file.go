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

func newFileJob(path, name string, fileInfo os.FileInfo) *FileJob {
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

	language, extension := DetectLanguage(name)

	if len(language) != 0 {
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
				return nil
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
				return nil
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
				return nil
			}
		}

		for _, l := range language {
			LoadLanguageFeature(l)
		}

		if !CountIgnore {
			for _, l := range language {
				if l == "ignore" || l == "gitignore" {
					return nil
				}
			}
		}

		return &FileJob{
			Location:          path,
			Symlocation:       symPath,
			Filename:          name,
			Extension:         extension,
			PossibleLanguages: language,
			Bytes:             fileInfo.Size(),
		}
	} else {
		printWarnF("skipping file unknown extension: %s", name)
	}

	return nil
}
