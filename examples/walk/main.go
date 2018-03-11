package main

import (
	"fmt"
	"github.com/MichaelTJones/walk"
	"os"
)

func main() {
	count := 0
	walk.Walk("./", func(root string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		count++

		return nil
	})

	fmt.Println(count)
}
