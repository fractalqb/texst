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
//	 .                             u uuuuuuuu uuuuuuuuuuuuuuuuuuuuuuuu
//	>   },
//	>   "origin": "10.0.0.1",
//	 +             aa a a a
//	>   "url": "https://httpbin.org/get"
//	> }
package texsting

import (
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
	t.Helper()
	return defaultConfig.Error(t, hint, subj)
}

func ErrorString(t *testing.T, hint, subj string) error {
	t.Helper()
	return defaultConfig.ErrorString(t, hint, subj)
}

func ErrorPipe(t *testing.T, hint string, subj func(io.Writer)) error {
	t.Helper()
	return defaultConfig.ErrorPipe(t, hint, subj)
}

func Fatal(t *testing.T, hint string, subj io.Reader) {
	t.Helper()
	defaultConfig.Fatal(t, hint, subj)
}

func FatalString(t *testing.T, hint, subj string) {
	t.Helper()
	defaultConfig.FatalString(t, hint, subj)
}

func FatalPipe(t *testing.T, hint string, subj func(io.Writer)) {
	t.Helper()
	defaultConfig.FatalPipe(t, hint, subj)
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
	t.Helper()
	if opts := recodTest(t); opts != nil {
		tcfg := cfg
		if opts.overwrite {
			tcfg.RecordOverwrite = true
		}
		tcfg.Record(t, hint, subj)
		return nil
	} else {
		msgs, err := cfg.compare(t, hint, subj)
		if err != nil {
			t.Fatal(err)
		}
		mmn := 0
		for _, msg := range msgs {
			switch msg[0] {
			case 'E':
				t.Error(msg[1:])
				mmn++
			case 'L':
				t.Log(msg[1:])
			default:
				t.Fatal(msg)
			}
		}
		if mmn > 0 {
			t.Errorf("%d mismatches", mmn)
		}
		return err
	}
}

func (cfg Config) ErrorString(t *testing.T, hint, subj string) error {
	t.Helper()
	return cfg.Error(t, hint, strings.NewReader(subj))
}

func (cfg Config) ErrorPipe(t *testing.T, hint string, subj func(io.Writer)) error {
	t.Helper()
	pr, pw := io.Pipe()
	go func() {
		t.Helper()
		subj(pw)
		pw.Close()
	}()
	return cfg.Error(t, hint, pr)
}

func (cfg Config) Fatal(t *testing.T, hint string, subj io.Reader) {
	t.Helper()
	if opts := recodTest(t); opts != nil {
		tcfg := cfg
		if opts.overwrite {
			tcfg.RecordOverwrite = true
		}
		tcfg.Record(t, hint, subj)
	} else {
		msgs, err := cfg.compare(t, hint, subj)
		if err != nil {
			t.Fatal(err)
		}
		for _, msg := range msgs {
			switch msg[0] {
			case 'E':
				t.Fatal(msg[1:])
			case 'L':
				t.Log(msg[1:])
			default:
				t.Fatal(msg)
			}
		}
	}
}

func (cfg Config) FatalString(t *testing.T, hint, subj string) {
	t.Helper()
	cfg.Fatal(t, hint, strings.NewReader(subj))
}

func (cfg Config) FatalPipe(t *testing.T, hint string, subj func(io.Writer)) {
	t.Helper()
	pr, pw := io.Pipe()
	go func() {
		t.Helper()
		subj(pw)
		pw.Close()
	}()
	cfg.Fatal(t, hint, pr)
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

func (cfg *Config) compare(t *testing.T, hint string, subj io.Reader) (msgs []string, err error) {
	t.Helper()
	cmpr := &texst.Texst{OnMismatch: mismatch(&msgs, hint)}
	if testing.Verbose() {
		cmpr.OnMatch = match(&msgs, hint)
	}
	reffile := cfg.RefFileName(t, hint)
	if _, err := os.Stat(reffile); os.IsNotExist(err) {
		t.Logf("to record a references file run '%[1]s=%[2]s go test -run %[2]s'",
			RecordEnv,
			t.Name(),
		)
		return nil, fmt.Errorf("reference texst file %s does not exists", reffile)
	}
	ref, err := texst.OpenRefFile(reffile)
	if err != nil {
		return nil, err
	}
	defer ref.Close()
	if !cfg.KeepSubject {
		_, err = cmpr.Check(ref, subj)
		return msgs, err
	}
	keepfile := reffile
	if filepath.Ext(keepfile) == ".texst" {
		keepfile = keepfile[:len(keepfile)-6]
	}
	k, err := os.CreateTemp(filepath.Dir(keepfile), filepath.Base(keepfile)+".texst-")
	if err != nil {
		return nil, err
	}
	defer func() {
		k.Close()
		if err == nil {
			os.Remove(k.Name())
		}
	}()
	_, err = cmpr.Check(ref, io.TeeReader(subj, k))
	return msgs, err
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
	if err = (texst.Prepare{}).Text(wr, subj); err != nil {
		t.Error(err)
	}
	t.Errorf("texst test-recorder wrote: %s", reffile)
}

func mismatch(t *[]string, hint string) texst.MismatchFunc {
	return func(n int, l []byte, ref []*texst.RefLine) {
		var sb strings.Builder
		if hint == "" {
			fmt.Fprintf(&sb, "Emismatch:%d [%s]", n, string(l))
		} else {
			fmt.Fprintf(&sb, "Emismatch %s:%d [%s]", hint, n, string(l))
		}
		for _, r := range ref {
			fmt.Fprintf(&sb, "\n%s:%d>%c[%s]",
				r.SourceName(),
				r.SourceLine(),
				r.IGroup(),
				r.Text(),
			)
		}
		*t = append(*t, sb.String())
	}
}

func match(t *[]string, hint string) texst.MatchFunc {
	return func(n int, l []byte, ref *texst.RefLine, match []int) {
		if hint == "" {
			hint = fmt.Sprintf("Lmatch:%d with %s:%d",
				n,
				ref.SourceName(),
				ref.SourceLine(),
			)
		} else {
			hint = fmt.Sprintf("Lmatch %s:%d with %s:%d",
				hint,
				n,
				ref.SourceName(),
				ref.SourceLine(),
			)
		}
		*t = append(*t, hint)
	}
}
