package main

import (
	"fmt"
	"os"

	"github.com/nitrocao/aliyun-workbench-cli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		if _, writeErr := fmt.Fprintln(os.Stderr, err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}
