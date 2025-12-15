package main

import (
	"flag"
	"fmt"
	"os"

	"gen-swagger-doc/pkg/driver/gin"
	"gen-swagger-doc/pkg/processor"
)

func main() {
	dir := flag.String("dir", ".", "Directory to scan")
	flag.Parse()

	absDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if *dir != "." {
		absDir = *dir
	}

	fmt.Printf("Scanning directory: %s\n", absDir)

	opts := processor.Options{
		WorkDir: absDir,
		Driver:  gin.NewDriver(),
	}

	if err := processor.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}
