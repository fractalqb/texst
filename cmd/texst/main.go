// A command line tool to use text tests
package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = struct {
	cobra.Command
	reffile string
	mlim    int
}{
	Command: cobra.Command{
		Use:   "texts",
		Short: "Compare a reference text file to subject files",
	},
}

func main() {
	rootCmd.Execute()
}
