package main

import (
	"fmt"
	"github.com/karrick/godirwalk"
)

func main() {
	count := 0
	godirwalk.Walk("./", &godirwalk.Options{
		Unsorted: true,
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			count++
			return nil
		},
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			return godirwalk.SkipNode
		},
	})
	fmt.Println(count)
}
