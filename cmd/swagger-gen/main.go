package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"gen-swagger-doc/pkg/driver"
	"gen-swagger-doc/pkg/driver/daenerys"
	"gen-swagger-doc/pkg/driver/gin"
	"gen-swagger-doc/pkg/processor"
)

func main() {
	dir := flag.String("dir", ".", "Directory to scan (absolute or relative path)")
	framework := flag.String("framework", "gin", "Framework to use: 'gin' or 'daenerys'")
	flag.Parse()

	// 1. Resolve absolute working directory
	workDir := *dir
	if workDir == "." {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting current working directory: %v\n", err)
			os.Exit(1)
		}
		workDir = wd
	}

	fmt.Printf("🚀 Starting Gen-Swagger-Doc\n")
	fmt.Printf("📂 Working Directory: %s\n", workDir)
	fmt.Printf("🔧 Framework: %s\n", *framework)

	// 2. Select Driver
	var d driver.Driver
	switch strings.ToLower(*framework) {
	case "gin":
		d = gin.NewDriver()
	case "daenerys":
		d = daenerys.New()
	default:
		fmt.Fprintf(os.Stderr, "❌ Error: Unsupported framework '%s'. Available: 'gin', 'daenerys'\n", *framework)
		os.Exit(1)
	}

	// 3. Run Processor
	opts := processor.Options{
		WorkDir: workDir,
		Driver:  d,
	}

	if err := processor.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Execution failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Done! Swagger comments injected successfully.")
}
