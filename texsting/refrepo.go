// Package texsting supports the use of texst in your Go tests.
//
// Example reads reference text from TestError.texst:
//
//	func TestError(t *testing.T) {
//		resp, _ := http.Get("https://httpbin.org/get")
//		defer resp.Body.Close()
//		Error(t, "", resp.Body)
//	}
//
// Reference Text:
//
//	> {
//	>   "args": {},
//	>   "headers": {
//	>     "Accept-Encoding": "gzip",
//	>     "Host": "httpbin.org",
//	>     "User-Agent": "Go-http-client/2.0",
//	 *                   aaaaaaaaaaaaaaaaaa
//	>     "X-Amzn-Trace-Id": "Root=1-602f798d-1c84bdc472ff9a2d3ec50f3b"
//	 =                             u uuuuuuuu uuuuuuuuuuuuuuuuuuuuuuuu
//	>   },
//	>   "origin": "10.0.0.1",
//	 +             aa a a a
//	>   "url": "https://httpbin.org/get"
//	> }
package texsting

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/fractalqb/texst"
)

func Error(t *testing.T, hint string, subj io.Reader) error {
	return defaultConfig.Error(t, hint, subj)
}

func Fatal(t *testing.T, hint string, subj io.Reader) {
	defaultConfig.Fatal(t, hint, subj)
}

func Record(t *testing.T, hint string, subj io.Reader) {
	defaultConfig.Record(t, hint, subj)
}

type RefRepo struct {
	Dir    string
	Suffix string
}

const (
	StdSuffix = ".texst"
	NoSuffix  = "\x00"
)

func (rr RefRepo) Filename(t *testing.T, hint string) string {
	suffix := rr.Suffix
	switch suffix {
	case "":
		suffix = StdSuffix
	case NoSuffix:
		suffix = ""
	}
	if hint == "" {
		return filepath.Join(rr.Dir, t.Name()+suffix)
	}
	if suffix == "" || strings.HasSuffix(hint, suffix) {
		return filepath.Join(rr.Dir, t.Name(), hint)
	}
	return filepath.Join(rr.Dir, t.Name(), hint+suffix)
}

type Config struct {
	RefFileName     func(t *testing.T, hint string) string
	MismatchLimit   int
	RecordOverwrite bool
}

var defaultConfig = Config{
	RefFileName:     RefRepo{Dir: "."}.Filename,
	MismatchLimit:   1,
	RecordOverwrite: false,
}

func (cfg Config) Error(t *testing.T, hint string, subj io.Reader) error {
	err := cfg.compare(t, hint, subj)
	if err != nil {
		t.Error(err)
	}
	return err
}

func (cfg Config) Fatal(t *testing.T, hint string, subj io.Reader) {
	err := cfg.compare(t, hint, subj)
	if err != nil {
		t.Fatal(err)
	}
}

func (cfg *Config) compare(t *testing.T, hint string, subj io.Reader) error {
	cmpr := &texst.Compare{
		MismatchLimit: cfg.MismatchLimit,
		OnMismatch:    MismatchError(t, hint, false),
	}
	reffile := cfg.RefFileName(t, hint)
	return cmpr.RefFile(reffile, subj)
}

func (cfg Config) Record(t *testing.T, hint string, subj io.Reader) {
	reffile := cfg.RefFileName(t, hint)
	if _, err := os.Stat(reffile); !os.IsNotExist(err) && !cfg.RecordOverwrite {
		t.Fatalf("TestRecord: reference file '%s' already exists", reffile)
	}
	dir := filepath.Dir(reffile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0777); err != nil {
			t.Fatal(err)
		}
	}
	wr, err := os.Create(reffile)
	if err != nil {
		t.Fatal(err)
	}
	defer wr.Close()
	scn := bufio.NewScanner(subj)
	for scn.Scan() {
		fmt.Fprintf(wr, "%c ", texst.TagRefLine)
		if _, err := wr.Write(scn.Bytes()); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintln(wr); err != nil {
			t.Fatal(err)
		}
	}
	t.Errorf("texst test-recorder wrote: %s", reffile)
}

func MismatchError(t *testing.T, hint string, abort bool) texst.MismatchFunc {
	if hint == "" {
		hint = "subject"
	}
	return func(ln int, l string, rln []*texst.RefLine) bool {
		lnstr := strconv.Itoa(ln)
		t.Errorf("%s:%s [%s]", hint, lnstr, l)
		padlen := utf8.RuneCountInString(hint) + len(lnstr)
		pad := strings.Repeat(" ", padlen)
		for _, r := range rln {
			t.Logf("%s%c [%s]", pad, r.IGroup(), r.Text())
		}
		return abort
	}
}
