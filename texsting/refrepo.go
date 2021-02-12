// Package texsting supports the use of texst in your Go tests.
package texsting

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.fractalqb.de/fractalqb/texst"
)

type RefRepo struct {
	Dir       string
	Suffix    string
	MissLimit int
}

func (rr *RefRepo) Compare(reffile string, subj io.Reader, onmiss texst.MissmatchFunc) error {
	cmpr := texst.Compare{MissmatchLimit: rr.MissLimit}
	rd, err := os.Open(reffile)
	if err != nil {
		return err
	}
	defer rd.Close()
	return cmpr.Readers(rd, subj, onmiss)
}

type missDesc struct {
	t   *testing.T
	ref string
}

func (dm missDesc) write(ln int, l string, rln []*texst.RefLine) bool {
	dm.t.Errorf("%s mismatch line %d [%s]", dm.ref, ln, l)
	for _, r := range rln {
		if r != nil {
			dm.t.Errorf("- ref '%c' [%s]", r.IGroup(), r.Text())
		}
	}
	return false
}

func (rr *RefRepo) TestError(t *testing.T, reffile string, subj io.Reader) error {
	ref := rr.reffile(t, reffile)
	err := rr.Compare(ref, subj, missDesc{t, reffile}.write)
	if err != nil {
		t.Error(err)
	}
	return err
}

func (rr *RefRepo) TestFatal(t *testing.T, reffile string, subj io.Reader) {
	ref := rr.reffile(t, reffile)
	err := rr.Compare(ref, subj, missDesc{t, reffile}.write)
	if err != nil {
		t.Fatal(err)
	}
}

func (rr *RefRepo) TestRecord(t *testing.T, reffile string, subj io.Reader) {
	if _, err := os.Stat(rr.Dir); err != nil {
		t.Fatal(err)
	}
	ref := rr.reffile(t, reffile)
	dir := filepath.Dir(ref)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0777); err != nil {
			t.Fatal(err)
		}
	}
	wr, err := os.Create(ref)
	if err != nil {
		t.Fatal(err)
	}
	defer wr.Close()
	scn := bufio.NewScanner(subj)
	for scn.Scan() {
		fmt.Fprintf(wr, "%c ", texst.TagRefLine)
		wr.Write(scn.Bytes())
		fmt.Fprintln(wr)
	}
	t.Errorf("texst test-recorder wrote: %s", ref)
}

func (rr *RefRepo) reffile(t *testing.T, reffile string) string {
	if reffile == "" {
		var suffix = rr.Suffix
		if suffix == "" {
			suffix = ".texst"
		}
		return t.Name() + suffix
	}
	if rr.Suffix == "" || strings.HasSuffix(reffile, rr.Suffix) {
		return filepath.Join(t.Name(), reffile)
	}
	return filepath.Join(t.Name(), reffile+rr.Suffix)
}
