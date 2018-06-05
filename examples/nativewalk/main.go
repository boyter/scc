package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	count := 0
	filepath.Walk("./", func(root string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		count++
		return nil
	})

	fmt.Println(count)
}
