package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	filepath.Walk("./", func(root string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fmt.Println(info.Name())
		return nil
	})
}
