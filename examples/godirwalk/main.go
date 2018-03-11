package main

import (
	"fmt"
	"github.com/karrick/godirwalk"
)

func main() {
	godirwalk.Walk("./", &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			fmt.Println(osPathname)
			return nil
		},
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			return godirwalk.SkipNode
		},
	})
}
