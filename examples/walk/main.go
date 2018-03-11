package main

import (
	"fmt"
	"github.com/MichaelTJones/walk"
	"os"
)

func main() {
	walk.Walk("./", func(root string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fmt.Println(info.Name())

		return nil
	})
}
