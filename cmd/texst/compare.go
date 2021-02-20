package main

import (
	"log"
	"os"

	"github.com/fractalqb/texst"
	"github.com/spf13/cobra"
)

func init() {
	compareCmd.Run = checkFiles
	compareCmd.Flags().StringVarP(&rootCmd.reffile, "reference", "r", "",
		"Set reference file name")
	compareCmd.MarkFlagRequired("reference")
	compareCmd.Flags().IntVarP(&rootCmd.mlim, "mismatch-limit", "l", 0,
		"Set the missmatch limit for comparison")
	rootCmd.AddCommand(&compareCmd.Command)
}

var compareCmd = struct {
	cobra.Command
	reffile string
	mlim    int
}{
	Command: cobra.Command{
		Use:   "compare",
		Short: "Compare a reference text file to subject files",
	},
}

func checkFiles(cmd *cobra.Command, files []string) {
	for _, f := range files {
		checkFile(rootCmd.reffile, f)
	}
}

func checkFile(ref, subj string) bool {
	sr, err := os.Open(subj)
	if err != nil {
		log.Fatal(err)
	}
	defer sr.Close()
	cmpr := texst.Compare{
		MismatchLimit: rootCmd.mlim,
		OnMismatch: func(n int, l string, rs []*texst.RefLine) bool {
			log.Printf("missmatch in line %d: '%s'\n", n, l)
			for _, r := range rs {
				log.Printf("- ref '%c':%s\n", r.IGroup(), r.Text())
			}
			return false
		},
	}
	err = cmpr.RefFile(ref, sr)
	if err == nil {
		log.Printf("%s matches reference %s\n", subj, ref)
		return true
	}
	log.Printf("%s mismatch with %s: %s", subj, ref, err)
	return false
}
