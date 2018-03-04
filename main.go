package main

import (
	"fmt"
	"github.com/monochromegane/go-gitignore"
	"github.com/ryanuber/columnize"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	// "runtime/debug"
	"strings"
	"sync"
)

type FileJob struct {
	Filename  string
	Extension string
	Location  string
	Content   []byte
	Count     int64
}

var Exclusions = strings.Split("woff,eot,cur,dm,xpm,emz,db,scc,idx,mpp,dot,pspimage,stl,dml,wmf,rvm,resources,tlb,docx,doc,xls,xlsx,ppt,pptx,msg,vsd,chm,fm,book,dgn,blines,cab,lib,obj,jar,pdb,dll,bin,out,elf,so,msi,nupkg,pyc,ttf,woff2,jpg,jpeg,png,gif,bmp,psd,tif,tiff,yuv,ico,xls,xlsx,pdb,pdf,apk,com,exe,bz2,7z,tgz,rar,gz,zip,zipx,tar,rpm,bin,dmg,iso,vcd,mp3,flac,wma,wav,mid,m4a,3gp,flv,mov,mp4,mpg,rm,wmv,avi,m4v,sqlite,class,rlib,ncb,suo,opt,o,os,pch,pbm,pnm,ppm,pyd,pyo,raw,uyv,uyvy,xlsm,swf", ",")

/// Get all the files that exist in the directory
func walkDirectory(root string, output *chan *FileJob) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		if !info.IsDir() {
			*output <- &FileJob{Location: path, Filename: info.Name()}
		}

		return nil
	})

	close(*output)
}

func fileReaderWorker(input *chan *FileJob, output *chan *FileJob) {
	var wg sync.WaitGroup
	for res := range *input {
		fileReadJob := res // Avoid race condition
		wg.Add(1)
		go func() {
			extension := path.Ext(fileReadJob.Filename)

			// TODO this should be a hashmap lookup for the speeds
			exclude := false
			for _, ex := range Exclusions {
				if strings.HasSuffix(fileReadJob.Filename, ex) {
					exclude = true
				}
			}

			if !exclude {
				content, _ := ioutil.ReadFile(fileReadJob.Location)
				fileReadJob.Content = content
				fileReadJob.Extension = extension
				*output <- fileReadJob
			}

			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(*output)
	}()
}

func fileProcessorWorker(input *chan *FileJob, output *chan *FileJob) {
	var wg sync.WaitGroup
	for res := range *input {
		fileReadJob := res // Avoid race condition
		// Do some pointless work
		wg.Add(1)
		go func() {
			count := 0
			count2 := 0
			for _, i := range fileReadJob.Content {
				if i == 0 {
					count++
				} else {
					count2++
				}
			}

			fileReadJob.Count = int64(count)

			*output <- fileReadJob
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(*output)
	}()
}

func fileSummeriser(input *chan *FileJob) {
	total := int64(0)
	count := 0

	languages := map[string]int64{}

	for res := range *input {

		// strings.Split(res.Filename, "sep") res.Filename

		if strings.HasSuffix(res.Filename, ".go") {
			_, ok := languages["Go"]

			if ok {
				languages["Go"] = languages["Go"] + 1
			} else {
				languages["Go"] = 1
			}

		}

		if strings.HasSuffix(res.Filename, ".yml") {
			_, ok := languages["Yaml"]

			if ok {
				languages["Yaml"] = languages["Yaml"] + 1
			} else {
				languages["Yaml"] = 1
			}

		}

		if strings.HasSuffix(res.Filename, ".toml") {
			_, ok := languages["TOML"]

			if ok {
				languages["TOML"] = languages["TOML"] + 1
			} else {
				languages["TOML"] = 1
			}

		}

		if strings.HasSuffix(res.Filename, ".md") {
			_, ok := languages["Markdown"]

			if ok {
				languages["Markdown"] = languages["Markdown"] + 1
			} else {
				languages["Markdown"] = 1
			}

		}

		// fmt.Println(res.Filename, res.Location, len(res.Content), res.Count)
		total += res.Count
		count++
	}

	for name, count := range languages {
		fmt.Println(name, count)
	}
}

//go:generate go run scripts/include.go
func main() {

	// A buffered channel that we can send work requests on.
	fileReadJobQueue := make(chan *FileJob, runtime.NumCPU()*100)
	fileProcessJobQueue := make(chan *FileJob, runtime.NumCPU()*2)
	fileSummaryJobQueue := make(chan *FileJob, runtime.NumCPU()*100)

	// debug.SetGCPercent(-1) // This can improve performance for some....

	go walkDirectory("/home/bboyter/Projects/redis/", &fileReadJobQueue)
	go fileReaderWorker(&fileReadJobQueue, &fileProcessJobQueue)
	go fileProcessorWorker(&fileProcessJobQueue, &fileSummaryJobQueue)
	fileSummeriser(&fileSummaryJobQueue) // Bring it all back to you

	// for res := range FileSummaryJobQueue {
	// 	fmt.Println(res.Location)
	// }

	// Once done lets print it all out
	output := []string{
		"Directory | File | License | Confidence | Size",
	}

	result := columnize.SimpleFormat(output)
	fmt.Println(result)

	// GitIgnore Processing
	gitignore, _ := gitignore.NewGitIgnore("./.gitignore")
	fmt.Println(gitignore.Match("./scc", false))
	fmt.Println(gitignore.Match("./LICENSE", false))
}
