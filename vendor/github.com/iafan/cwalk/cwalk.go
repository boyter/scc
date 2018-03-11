package cwalk

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// NumWorkers defines how many workers to run
// on each Walk() function invocation
var NumWorkers = runtime.GOMAXPROCS(0)

// BufferSize defines the size of the job buffer
var BufferSize = NumWorkers

// ErrNotDir indicates that the path, which is being passed
// to a walker function, does not point to a directory
var ErrNotDir = errors.New("Not a directory")

// Walker is constructed for each Walk() function invocation
type Walker struct {
	wg       sync.WaitGroup
	jobs     chan string
	walkFunc filepath.WalkFunc
}

// the readDirNames function below was taken from the original
// implementation (see https://golang.org/src/path/filepath/path.go)
// but has sorting removed (sorting doesn't make sense
// in concurrent execution, anyway)

// readDirNames reads the directory named by dirname and returns
// a list of directory entries.
func readDirNames(dirname string) ([]string, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	return names, nil
}

// processPath processes one directory and adds
// its subdirectories to the queue for further processing
func (w *Walker) processPath(path string) error {
	defer w.wg.Done()

	names, err := readDirNames(path)
	if err != nil {
		return err
	}

	root := path
	for _, name := range names {
		path = filepath.Join(root, name)
		info, err := os.Lstat(path)
		err = w.walkFunc(path, info, err)

		if err == nil && info.IsDir() {
			w.addJob(path)
		}
		if err != nil && err != filepath.SkipDir {
			return err
		}
	}
	return nil
}

// addJob increments the job counter
// and pushes the path to the jobs channel
func (w *Walker) addJob(path string) {
	w.wg.Add(1)
	select {
	// try to push the job to the channel
	case w.jobs <- path: // ok
	default: // buffer overflow
		// process job synchronously
		w.processPath(path)
	}
}

// worker processes all the jobs
// until the jobs channel is explicitly closed
func (w *Walker) worker() {
	for path := range w.jobs {
		w.processPath(path)
	}
}

// Walk recursively descends into subdirectories,
// calling walkFn for each file or directory
// in the tree, including root directory.
// Walk does not follow symbolic links.
func (w *Walker) Walk(root string, walkFn filepath.WalkFunc) error {
	w.jobs = make(chan string, BufferSize)
	w.walkFunc = walkFn

	info, err := os.Lstat(root)
	err = w.walkFunc(root, info, err)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return ErrNotDir
	}

	// spawn workers
	for n := 1; n <= NumWorkers; n++ {
		go w.worker()
	}
	w.addJob(root) // add root path as a first job
	w.wg.Wait()    // wait till all paths are processed
	close(w.jobs)  // signal workers to close

	return nil
}

// Walk is a wrapper function for the Walker object
// that mimicks the behavior of filepath.Walk
func Walk(root string, walkFn filepath.WalkFunc) error {
	w := Walker{}
	return w.Walk(root, walkFn)
}
