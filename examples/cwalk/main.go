package main

import (
	"fmt"
	"github.com/iafan/cwalk"
	"os"
)

func main() {
	cwalk.Walk("./", func(root string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fmt.Println(info.Name())
		return nil
	})
}
