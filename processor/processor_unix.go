package processor

import (
	"syscall"
)

func init() {
	// Doing the limits configuration in init() directly is possible, but it
	// means that we wouldn't have access to user flags like Verbose.
	ConfigureLimits = ConfigureLimitsUnix
}

func ConfigureLimitsUnix() {
	margin := 16

	// High water mark of how many open files we may need.
	// Each FileReadJobWorker needs one + DirectoryWorker may need up to one per runtime CPU
	// We pad it out a bit because every process needs a few handles just to exist
	highWaterMark := uint64(FileProcessJobWorkers + DirectoryWalkerJobWorkers + margin)

	limit := syscall.Rlimit{}
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit)
	if err != nil {
		printWarnf("Unable to determine open file limit: %v", err)
		return
	}

	originalLimit := limit.Cur

	printDebugf("Open file limit: current=%d max=%d", limit.Cur, limit.Max)

	// limit.Cur is the current (soft) limit. If this is too low, we may be
	// able to raise it with Setrlimit
	if limit.Cur < highWaterMark {
		limit.Cur = highWaterMark

		// limit.Max is the hard limit. If this is still too low, we'll scale
		// it as high as we can but we also have to scale back how many workers
		// we launch.
		if limit.Max < highWaterMark {
			printWarn("Scaling down workers to fit open file ulimit - performance may be sub-optimal")
			limit.Cur = limit.Max
			scaleWorkersToLimit(int(limit.Max), margin)
		}

		if originalLimit < limit.Max {
			err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit)
			if err != nil {
				printWarnf("Adjusting file limit failed: %v", err)
				scaleWorkersToLimit(int(originalLimit), margin)
			} else {
				printDebugf("Adjusted open file limit to %d", limit.Cur)
			}
		}
	}
}

func scaleWorkersToLimit(max, margin int) {
	scalingBase := (max - margin) / 5

	DirectoryWalkerJobWorkers = scalingBase
	FileProcessJobWorkers = scalingBase * 4
}
