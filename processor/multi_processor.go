package processor

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
)

type MultiProcessor struct {
	workQueue      chan ProcessInput
	processedCount int32
	wg             sync.WaitGroup
}

type ProcessInput struct {
	Dir      string
	OnResult func(stat LocStat)
}

type LocStat struct {
	Files, Lines, Code, Comment, Blank, Complexity int64
}

func (loc LocStat) String() string {
	return fmt.Sprintf("Loc Stat: %d files, %d lines, %d code, %d comment, %d blank, %d complexity",
		loc.Files, loc.Lines, loc.Code, loc.Comment, loc.Blank, loc.Complexity)
}

func (pi ProcessInput) String() string {
	return fmt.Sprintf("DIR: %s", pi.Dir)
}

func NewMultiProcessor(inParallel int) *MultiProcessor {
	mp := &MultiProcessor{
		workQueue: make(chan ProcessInput),
	}

	mp.SetupProcessor()

	mp.wg.Add(inParallel)

	for i := 1; i <= inParallel; i++ {
		go func(i int) {
			log.Infof("Started MP worker %d", i)
			for pi := range mp.workQueue {
				mp.process(pi)
			}
			log.Infof("MP worker %d is done!", i)
			mp.wg.Done()
		}(i)
	}

	return mp
}

func (mp *MultiProcessor) Process(dir string, onResult func(stat LocStat)) {
	input := ProcessInput{Dir: dir, OnResult: onResult}
	log.Infof("Queue processing %v", input)
	mp.workQueue <- input
}

func (mp *MultiProcessor) Complete() {
	log.Infof("Completing MP")
	close(mp.workQueue)
	mp.wg.Wait()
	log.Infof("!!!MultiProcessor is finished")
}

func (mp *MultiProcessor) process(input ProcessInput) {
	processedNo := atomic.AddInt32(&mp.processedCount, 1)
	log.Infof("Start processing %d: %v", processedNo, input.Dir)

	fileListQueue := make(chan *FileJob, FileListQueueSize)                     // Files ready to be read from disk
	fileReadContentJobQueue := make(chan *FileJob, FileReadContentJobQueueSize) // Files ready to be processed
	fileSummaryJobQueue := make(chan *FileJob, FileSummaryJobQueueSize)         // Files ready to be summerised

	go walkDirectoryParallel(input.Dir, fileListQueue)
	go fileReaderWorker(fileListQueue, fileReadContentJobQueue)
	go fileProcessorWorker(fileReadContentJobQueue, fileSummaryJobQueue)

	output := mp.summarize(fileSummaryJobQueue)
	log.Infof("Completed processing %d: %v: %v", processedNo, input, output)
	input.OnResult(output)
}

func (mp *MultiProcessor) SetupProcessor() {
	ConfigureGc()
	ProcessConstants()
}

func (mp *MultiProcessor) summarize(input chan *FileJob) (output LocStat) {
	for res := range input {
		output.Files++
		output.Lines += res.Lines
		output.Code += res.Code
		output.Comment += res.Comment
		output.Blank += res.Blank
		output.Complexity += res.Complexity
	}
	return
}
