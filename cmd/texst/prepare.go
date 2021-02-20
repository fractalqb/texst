package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/fractalqb/texst/texsting"
)

func init() {
	prepareCmd.Run = prepareFiles
	prepareCmd.Flags().StringVarP(
		&prepareCmd.suffix,
		"suffix", "s",
		prepareCmd.suffix,
		"Set file suffix for created reference text files")
	prepareCmd.Flags().BoolVarP(
		&prepareCmd.force,
		"force", "f",
		prepareCmd.force,
		"Force to overwrite existing reference files")
	rootCmd.AddCommand(&prepareCmd.Command)
}

var prepareCmd = struct {
	cobra.Command
	suffix string
	force  bool
}{
	Command: cobra.Command{
		Use:   "prepare",
		Short: "Prepare basic reference text file",
	},
	suffix: texsting.StdSuffix,
	force:  false,
}

func prepareFiles(cmd *cobra.Command, files []string) {
	if len(files) == 0 {
		prepare(os.Stdin, os.Stdout)
	} else {
		for _, f := range files {
			prepareFile(f)
		}
	}
}

func prepare(rd io.Reader, wr io.Writer) {
	prefix := []byte{'>', ' '}
	scn := bufio.NewScanner(rd)
	for scn.Scan() {
		wr.Write(prefix)
		wr.Write(scn.Bytes())
		fmt.Fprintln(wr)
	}
}

func prepareFile(name string) {
	texstfile := name + ".texst"
	if _, err := os.Stat(texstfile); !os.IsNotExist(err) {
		if !prepareCmd.force {
			log.Fatalf("%s already exists", texstfile)
		}
	}
	rd, _ := os.Open(name)
	defer rd.Close()
	wr, _ := os.Create(texstfile)
	defer wr.Close()
	prepare(rd, wr)
}
