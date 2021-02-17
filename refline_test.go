package texst

import (
	"bufio"
	"reflect"
	"strings"
	"testing"

	"git.fractalqb.de/fractalqb/icontainer/islist"
)

var _ islist.Node = (*RefLine)(nil)

func TestSegment_sub(t *testing.T) {
	testCase := func(s, e, exps, expe, splits, splite int) func(*testing.T) {
		return func(t *testing.T) {
			seg := mask{start: 2, end: 5}
			split := seg.sub(&mask{start: s, end: e})
			if seg.start != exps {
				t.Errorf("start: expect %d / got %d", exps, seg.start)
			}
			if seg.end != expe {
				t.Errorf("end: expect %d / got %d", expe, seg.end)
			}
			if splits < 0 {
				if splite >= 0 {
					t.Fatal("inconsistent split expectation")
				}
				if split != nil {
					t.Errorf("unexpected split [%d;%d)", split.start, split.end)
				}
				return
			} else if splite < 0 {
				t.Fatal("inconsistent split expectation")
			}
			if split.start != splits {
				t.Errorf("start: expect %d / got %d", splits, split.start)
			}
			if split.end != splite {
				t.Errorf("end: expect %d / got %d", splite, split.end)
			}
		}
	}
	t.Run("before with gap", testCase(0, 1, 2, 5, -1, -1))
	t.Run("before touch", testCase(0, 2, 2, 5, -1, -1))
	t.Run("cut start", testCase(1, 3, 3, 5, -1, -1))
	t.Run("head", testCase(2, 4, 4, 5, -1, -1))
	t.Run("match", testCase(2, 5, 2, 2, -1, -1))
	t.Run("split", testCase(3, 4, 2, 3, 4, 5))
	t.Run("cut end", testCase(4, 6, 2, 4, -1, -1))
	t.Run("after touch", testCase(5, 7, 2, 5, -1, -1))
	t.Run("after with gap", testCase(6, 7, 2, 5, -1, -1))
}

func TestRefLine_lineSegs(t *testing.T) {
	var rl RefLine
	rl.masksPattern(" xy xx  zzzz", ArgMaskExact)
	rl.masksPattern("a  bbaa  cc", ArgMaskExact)
	expect := []mask{
		{name: 'a', start: 0, end: 1, mode: ArgMaskExact},
		{name: 'x', start: 1, end: 2, mode: ArgMaskExact},
		{name: 'y', start: 2, end: 3, mode: ArgMaskExact},
		{name: 'b', start: 3, end: 5, mode: ArgMaskExact},
		{name: 'a', start: 5, end: 7, mode: ArgMaskExact},
		{name: 'z', start: 8, end: 9, mode: ArgMaskExact},
		{name: 'c', start: 9, end: 11, mode: ArgMaskExact},
		{name: 'z', start: 11, end: 12, mode: ArgMaskExact},
	}
	eq := reflect.DeepEqual(rl.masks, expect)
	if !eq {
		t.Fatalf("wrong line segments:\n%+v\n%+v", expect, rl.masks)
	}
}

func TestRefLine_Read(t *testing.T) {
	var lno int
	t.Run("wrong tag on line", func(t *testing.T) {
		var rl RefLine
		err := rl.read(bufio.NewReader(strings.NewReader("x\n>")), "", &lno)
		if err == nil {
			t.Fatal("accepted non-reference-line")
		}
		if !strings.HasPrefix(err.Error(), "syntax error: not a ref") {
			t.Fatalf("wrong error: %s", err)
		}
	})
	t.Run("wrong tag on last line", func(t *testing.T) {
		var rl RefLine
		err := rl.read(bufio.NewReader(strings.NewReader("x")), "", &lno)
		if err == nil {
			t.Fatal("accepted non-reference-line")
		}
		if !strings.HasPrefix(err.Error(), "syntax error: not a ref") {
			t.Fatalf("wrong error: %s", err)
		}
	})
	t.Run("no igroup in line", func(t *testing.T) {
		var rl RefLine
		err := rl.read(bufio.NewReader(strings.NewReader(">\n> ")), "", &lno)
		if err == nil {
			t.Fatal("accepted line without igroup")
		}
		if !strings.HasPrefix(err.Error(), "syntax error: incomplete reference line") {
			t.Fatalf("wrong error: %s", err)
		}
	})
	t.Run("no igroup in last line", func(t *testing.T) {
		var rl RefLine
		err := rl.read(bufio.NewReader(strings.NewReader(">")), "", &lno)
		if err == nil {
			t.Fatal("accepted line without igroup")
		}
		if !strings.HasPrefix(err.Error(), "syntax error: incomplete reference line") {
			t.Fatalf("wrong error: %s", err)
		}
	})
	t.Run("valid line", func(t *testing.T) {
		var rl RefLine
		err := rl.read(bufio.NewReader(strings.NewReader(">:this is content\n")), "", &lno)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if rl.igroup != ':' {
			t.Errorf("unexpected igroup '%c', want ':'", rl.igroup)
		}
		if rl.text != "this is content" {
			t.Errorf("wrong content: '%s'", rl.text)
		}
	})
	t.Run("valid last line", func(t *testing.T) {
		var rl RefLine
		err := rl.read(bufio.NewReader(strings.NewReader(">:this is content")), "", &lno)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if rl.igroup != ':' {
			t.Errorf("unexpected igroup '%c', want ':'", rl.igroup)
		}
		if rl.text != "this is content" {
			t.Errorf("wrong content: '%s'", rl.text)
		}
	})
	t.Run("valid with segments", func(t *testing.T) {
		var rl RefLine
		err := rl.read(bufio.NewReader(strings.NewReader(
			">:just example content\n =  xx         yyyyyyy",
		)), "", &lno)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		expect := []mask{
			{
				name:  'x',
				mode:  ArgMaskExact,
				start: 2, end: 4,
				refStart: 2, refEnd: 4,
			},
			{
				name:  'y',
				mode:  ArgMaskExact,
				start: 13, end: 20,
				refStart: 13, refEnd: 20,
			},
		}
		if !reflect.DeepEqual(rl.masks, expect) {
			t.Fatalf("wrong line segments:\n%+v\n%+v", expect, rl.masks)
		}
	})
}

func TestRefLine_matches(t *testing.T) {
	testCase := func(caseName, ref, subj string, match bool) {
		t.Run(caseName, func(t *testing.T) {
			var lno int
			rl := newRefLine()
			err := rl.read(bufio.NewReader(strings.NewReader(ref)), "", &lno)
			if err != nil {
				t.Fatal(err)
			}
			err = rl.matches(subj)
			if (err == nil) != match {
				if match {
					t.Errorf(`unexpected mismatch: %s
  %s
%s`,
						err, subj, ref)
				} else {
					t.Errorf(`unexpected match:
  %s
%s`,
						subj, ref)
				}
			} else if err != nil {
				t.Logf("expected: %s", err)
			}
		})
	}
	testCase("verbatim match",
		"> abcdef世界ijklmnopqrstuvwxyz",
		"abcdef世界ijklmnopqrstuvwxyz",
		true,
	)

	testCase("seg exact prefix match", `#
> abcdef世界ijklmnopqrstuvwxyz
 =---`,
		"XXXdef世界ijklmnopqrstuvwxyz",
		true,
	)
	testCase("seg exact prefix mismatch", `#
> abcdef世界ijklmnopqrstuvwxyz
 =---`,
		"abcXef世界ijklmnopqrstuvwxyz",
		false,
	)
	testCase("seg exact suffix match", `#
> abcdef世界ijklmnopqrstuvwxyz
 =                       ---`,
		"abcdef世界ijklmnopqrstuvwXXX",
		true,
	)
	testCase("seg exact suffix mismatch", `#
> abcdef世界ijklmnopqrstuvwxyz
 =                       ---`,
		"abcdef世界ijklmnopqrstuvXxyz",
		false,
	)
	testCase("seg exact mid match", `#
> abcdef世界ijklmnopqrstuvwxyz
 =            --`,
		"abcdef世界ijklmnopqrstuvwxyz",
		true,
	)
	testCase("seg exact mid mismatch left", `#
> abcdef世界ijklmnopqrstuvwxyz
 =            --`,
		"abcdef世界ijkXmnopqrstuvwxyz",
		false,
	)
	testCase("seg exact mid match right", `#
> abcdef世界ijklmnopqrstuvwxyz
 =            --`,
		"abcdef世界ijklmnXpqrstuvwxyz",
		false,
	)

	testCase("seg opt prefix match", `#
> abcdef世界ijklmnopqrstuvwxyz
 *---`,
		"XXXdef世界ijklmnopqrstuvwxyz",
		true,
	)
	testCase("seg opt prefix mismatch", `#
> abcdef世界ijklmnopqrstuvwxyz
 *---`,
		"abcXef世界ijklmnopqrstuvwxyz",
		false,
	)
	testCase("seg opt suffix match", `#
> abcdef世界ijklmnopqrstuvwxyz
 *                       ---`,
		"abcdef世界ijklmnopqrstuvwXXX",
		true,
	)
	testCase("seg opt suffix mismatch", `#
> abcdef世界ijklmnopqrstuvwxyz
 *                       ---`,
		"abcdef世界ijklmnopqrstuvXxyz",
		false,
	)
	testCase("seg opt mid match", `#
> abcdef世界ijklmnopqrstuvwxyz
 *            --`,
		"abcdef世界ijklmnopqrstuvwxyz",
		true,
	)
	testCase("seg opt mid mismatch left", `#
> abcdef世界ijklmnopqrstuvwxyz
 *            --`,
		"abcdef世界ijkXmnopqrstuvwxyz",
		false,
	)
	testCase("seg opt mid mismatch right", `#
> abcdef世界ijklmnopqrstuvwxyz
 *            --`,
		"abcdef世界ijklmnXpqrstuvwxyz",
		false,
	)
	testCase("seg opt backtrack match", `#
> aXXbYYc
 * xx
 =    yy`,
		"a.bb..c",
		true)
	testCase("seg opt backtrack mismatch", `#
> aXXbYYc
 * xx
 =    yy`,
		"a.bb..C",
		false)

	testCase("seg var prefix match", `#
> abcdef世界ijklmnopqrstuvwxyz
 +---`,
		"XXXdef世界ijklmnopqrstuvwxyz",
		true,
	)
	testCase("seg var prefix mismatch", `#
> abcdef世界ijklmnopqrstuvwxyz
 +---`,
		"abcXef世界ijklmnopqrstuvwxyz",
		false,
	)
	testCase("seg var suffix match", `#
> abcdef世界ijklmnopqrstuvwxyz
 +                       ---`,
		"abcdef世界ijklmnopqrstuvwXXX",
		true,
	)
	testCase("seg var suffix mismatch", `#
> abcdef世界ijklmnopqrstuvwxyz
 +                       ---`,
		"abcdef世界ijklmnopqrstuvXxyz",
		false,
	)
	testCase("seg var mid match", `#
> abcdef世界ijklmnopqrstuvwxyz
 +            --`,
		"abcdef世界ijklmnopqrstuvwxyz",
		true,
	)
	testCase("seg var mid mismatch left", `#
> abcdef世界ijklmnopqrstuvwxyz
 +            --`,
		"abcdef世界ijkXmnopqrstuvwxyz",
		false,
	)
	testCase("seg var mid mismatch right", `#
> abcdef世界ijklmnopqrstuvwxyz
 +            --`,
		"abcdef世界ijklmnXpqrstuvwxyz",
		false,
	)
	testCase("seg var backtrack match", `#
> aXXbYYc
 + xx
 =    yy`,
		"a.bb..c",
		true)
	testCase("seg var backtrack mismatch", `#
> aXXbYYc
 + xx
 =    yy`,
		"a.bb..C",
		false)

	testCase("seg opt empty match", `#
> abcdefg
 *  ---`,
		"abfg",
		true)
	testCase("seg var empty mismatch", `#
> abcdefg
 +  ---`,
		"abfg",
		false)
}
