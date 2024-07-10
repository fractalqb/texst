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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/fractalqb/texst"
)

// When this environment variable is set to a regexp and the name of the current
// test matches calls to Error or Fatal will record the subj as new reference
// data instead of comparing it. E.g.
//
//	TEXSTING_RECORD=TestRecording go test .
const RecordEnv = "TEXSTING_RECORD"

// GoTestdataDir is the name of Go's default directory for testdata (see go help
// test).
const GoTestdataDir = "testdata"

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
	KeepSubject     bool
}

var defaultConfig = Config{
	RefFileName:     RefRepo{Dir: GoTestdataDir}.Filename,
	MismatchLimit:   1,
	RecordOverwrite: false,
	KeepSubject:     true,
}

func (cfg Config) Error(t *testing.T, hint string, subj io.Reader) error {
	if opts := recodTest(t); opts != nil {
		tcfg := cfg
		if opts.overwrite {
			tcfg.RecordOverwrite = true
		}
		tcfg.Record(t, hint, subj)
		return nil
	} else {
		err := cfg.compare(t, hint, subj)
		if err != nil {
			t.Error(err)
		}
		return err
	}
}

func (cfg Config) Fatal(t *testing.T, hint string, subj io.Reader) {
	if opts := recodTest(t); opts != nil {
		tcfg := cfg
		if opts.overwrite {
			tcfg.RecordOverwrite = true
		}
		tcfg.Record(t, hint, subj)
	} else {
		err := cfg.compare(t, hint, subj)
		if err != nil {
			t.Fatal(err)
		}
	}
}

type recordOpts struct {
	overwrite bool
}

func recodTest(t *testing.T) *recordOpts {
	rec := os.Getenv(RecordEnv)
	if rec == "" {
		return nil
	}
	recs := strings.Split(rec, " ")
	r, err := regexp.Compile(recs[len(recs)-1])
	if err != nil {
		t.Logf("texsting: invalid regexp '%s' in %s, not recording: %s", rec, RecordEnv, err)
		return nil
	}
	var opts recordOpts
	for _, o := range recs[:len(recs)-1] {
		switch o {
		case "f", "force":
			opts.overwrite = true
		default:
			t.Logf("texsting: invalid recording option '%s'", o)
		}
	}
	if r.MatchString(t.Name()) {
		return &opts
	}
	return nil
}

func (cfg *Config) compare(t *testing.T, hint string, subj io.Reader) (err error) {
	cmpr := &texst.Texst{OnMismatch: MismatchError(t, hint, false)}
	if testing.Verbose() {
		cmpr.OnMatch = MatchLog(t, hint)
	}
	reffile := cfg.RefFileName(t, hint)
	if _, err := os.Stat(reffile); os.IsNotExist(err) {
		t.Logf("to record a references file run '%[1]s=%[2]s go test -run %[2]s'",
			RecordEnv,
			t.Name(),
		)
		return fmt.Errorf("reference texst file %s does not exists", reffile)
	}
	ref, err := texst.OpenRefFile(reffile)
	if err != nil {
		return err
	}
	defer ref.Close()
	if !cfg.KeepSubject {
		return cmpr.Check(ref, subj)
	}
	keepfile := reffile
	if filepath.Ext(keepfile) == ".texst" {
		keepfile = keepfile[:len(keepfile)-6]
	}
	k, err := os.CreateTemp(filepath.Dir(keepfile), filepath.Base(keepfile)+".texst-")
	if err != nil {
		return err
	}
	defer func() {
		k.Close()
		if err == nil {
			os.Remove(k.Name())
		}
	}()
	return cmpr.Check(ref, io.TeeReader(subj, k))
}

func (cfg Config) Record(t *testing.T, hint string, subj io.Reader) {
	reffile := cfg.RefFileName(t, hint)
	if !cfg.RecordOverwrite {
		if _, err := os.Stat(reffile); err == nil || !os.IsNotExist(err) {
			t.Fatalf("TestRecord: reference file '%s' already exists", reffile)
		}
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
	if err = texst.Prepare(wr, subj); err != nil {
		t.Error(err)
	}
	t.Errorf("texst test-recorder wrote: %s", reffile)
}

func MismatchError(t *testing.T, hint string, abort bool) texst.MismatchFunc {
	if hint == "" {
		hint = t.Name()
	}
	return func(n int, l []byte, ref []*texst.RefLine) (err error) {
		var sb strings.Builder
		fmt.Fprintf(&sb, "mismatch %s:%d [%s]", hint, n, string(l))
		subj := sb.String()
		sb.WriteString(" ref:")
		for _, r := range ref {
			fmt.Fprintf(&sb, " %d", r.SourceLine())
		}
		err = errors.New(sb.String())
		sb.Reset()
		sb.WriteString(subj)
		for _, r := range ref {
			fmt.Fprintf(&sb, "\n%s:%d>%c[%s]",
				r.SourceName(),
				r.SourceLine(),
				r.IGroup(),
				r.Text(),
			)
		}
		t.Error(sb.String())
		if abort {
			return err
		}
		return nil
	}
}

func MatchLog(t *testing.T, hint string) texst.MatchFunc {
	if hint == "" {
		hint = t.Name()
	}
	return func(n int, l []byte, ref *texst.RefLine, match []int) error {
		if hint == "" {
			hint = ref.SourceName()
		}
		t.Logf("match %s:%d with %s:%d", hint, n, ref.SourceName(), ref.SourceLine())
		return nil
	}
}
