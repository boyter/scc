package main

import (
	"io/ioutil"
	"testing"
)

func BenchmarkIoUtilRead10k(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		by, _ := ioutil.ReadFile("./10k")
		count = len(by)
	}
	b.Log(count)
}

func BenchmarkBuffIoRead10k32768(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		count = len(bufferedReadFile("./10k", 32768))
	}
	b.Log(count)
}

func BenchmarkIoUtilRead100k(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		by, _ := ioutil.ReadFile("./100k")
		count = len(by)
	}
	b.Log(count)
}

func BenchmarkBuffIoRead100k32768(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		count = len(bufferedReadFile("./100k", 32768))
	}
	b.Log(count)
}

func BenchmarkIoUtilRead1000k(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		by, _ := ioutil.ReadFile("./1000k")
		count = len(by)
	}
	b.Log(count)
}

func BenchmarkBuffIoRead1000k32768(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		count = len(bufferedReadFile("./1000k", 32768))
	}
	b.Log(count)
}

func BenchmarkIoUtilReadLinuxAverage(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		by, _ := ioutil.ReadFile("./linuxaverage")
		count = len(by)
	}
	b.Log(count)
}

func BenchmarkBuffIoReadLinuxAverage32768(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		count = len(bufferedReadFile("./linuxaverage", 32768))
	}
	b.Log(count)
}

func BenchmarkIoUtilReadText(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		by, _ := ioutil.ReadFile("./textfile.json")
		count = len(by)
	}
	b.Log(count)
}

func BenchmarkBuffIoReadText32768(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		count = len(bufferedReadFile("./textfile.json", 32768))
	}
	b.Log(count)
}
