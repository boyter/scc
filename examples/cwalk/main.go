package main

import (
	"fmt"
	"github.com/iafan/cwalk"
	"os"
)

func main() {
	count := 0
	cwalk.Walk("./", func(root string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		count++
		return nil
	})
	fmt.Println(count)
}
