package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	// "sync"
)

func bufferRead(file string, buffersize int) []byte {
	data := []byte{}

	fi, err := os.Open(file)
	if err != nil {
		panic(err)
	}

	// make a read buffer
	r := bufio.NewReader(fi)

	// make a buffer to keep chunks that are read
	buf := make([]byte, buffersize)
	for {
		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			fi.Close()
			panic(err)
		}
		if n == 0 {
			break
		} else {
			fmt.Println("///////////////////////////////////////////////")
			fmt.Println(string(buf))

			data = append(data, buf...)
		}
	}

	fmt.Println("///////////////////////////////////////////////")
	fi.Close()
	return data
}

// func parallel() {

// 	const BufferSize = 100
// 	file, err := os.Open("./main.go")
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	defer file.Close()

// 	fileinfo, err := file.Stat()
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}

// 	filesize := int(fileinfo.Size())
// 	// Number of go routines we need to spawn.
// 	concurrency := filesize / BufferSize

// 	// check for any left over bytes. Add one more go routine if required.
// 	if remainder := filesize % BufferSize; remainder != 0 {
// 		concurrency++
// 	}

// 	var wg sync.WaitGroup
// 	wg.Add(concurrency)

// 	for i := 0; i < concurrency; i++ {
// 		go func(chunksizes []chunk, i int) {
// 			defer wg.Done()

// 			chunk := chunksizes[i]
// 			buffer := make([]byte, chunk.bufsize)
// 			bytesread, err := file.ReadAt(buffer, chunk.offset)

// 			// As noted above, ReadAt differs slightly compared to Read when the
// 			// output buffer provided is larger than the data that's available
// 			// for reading. So, let's return early only if the error is
// 			// something other than an EOF. Returning early will run the
// 			// deferred function above
// 			if err != nil && err != io.EOF {
// 				fmt.Println(err)
// 				return
// 			}

// 			fmt.Println("bytes read, string(bytestream): ", bytesread)
// 			fmt.Println("bytestream to string: ", string(buffer[:bytesread]))
// 		}(chunksizes, i)
// 	}

// 	wg.Wait()

// }

func bufferedReadFile(fileLocation string, buffersize int) []byte {

	file, err := os.Open(fileLocation)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer file.Close()

	output := []byte{}
	buffer := make([]byte, buffersize)

	for {
		bytesread, err := file.Read(buffer)

		if err != nil {
			if err != io.EOF {
				fmt.Println(err)
			}

			break
		}

		output = append(output, buffer[:bytesread]...)
	}

	return output
}

func main() {
	// data := bufferRead("./main.go", 300)
	// fmt.Println(len(data))
	// res := bufferedReadFile("./textfile.json", 300000)
	res, _ := ioutil.ReadFile("./textfile.json")
	fmt.Println(string(res))
	// parallel()
}
