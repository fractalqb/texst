package texst

import (
	"bufio"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"git.fractalqb.de/fractalqb/icontainer/islist"
)

var _ islist.Node = (*RefLine)(nil)

func TestSegment_sub(t *testing.T) {
	testCase := func(s, e, exps, expe, splits, splite int) func(*testing.T) {
		return func(t *testing.T) {
			seg := segment{start: 2, end: 5}
			split := seg.sub(&segment{start: s, end: e})
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
	rl.lineSegs(" xy xx  zzzz", ArgSegExact)
	rl.lineSegs("a  bbaa  cc", ArgSegExact)
	expect := []segment{
		{name: 'a', start: 0, end: 1, mode: ArgSegExact},
		{name: 'x', start: 1, end: 2, mode: ArgSegExact},
		{name: 'y', start: 2, end: 3, mode: ArgSegExact},
		{name: 'b', start: 3, end: 5, mode: ArgSegExact},
		{name: 'a', start: 5, end: 7, mode: ArgSegExact},
		{name: 'z', start: 8, end: 9, mode: ArgSegExact},
		{name: 'c', start: 9, end: 11, mode: ArgSegExact},
		{name: 'z', start: 11, end: 12, mode: ArgSegExact},
	}
	eq := reflect.DeepEqual(rl.segs, expect)
	if !eq {
		t.Fatalf("wrong line segments:\n%+v\n%+v", expect, rl.segs)
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
		expect := []segment{
			{
				name:  'x',
				mode:  ArgSegExact,
				start: 2, end: 4,
				refStart: 2, refEnd: 4,
			},
			{
				name:  'y',
				mode:  ArgSegExact,
				start: 13, end: 20,
				refStart: 13, refEnd: 20,
			},
		}
		if !reflect.DeepEqual(rl.segs, expect) {
			t.Fatalf("wrong line segments:\n%+v\n%+v", expect, rl.segs)
		}
	})
}

func ExampleRefLine_preSegPart() {
	rl := RefLine{text: "Hello, 世界!"}
	rl.addSegment(' ', ArgSegExact, 2, 4)
	rl.addSegment(' ', ArgSegExact, 7, 9)
	fmt.Printf("[%s]\n", rl.preSegPart(0))
	fmt.Printf("[%s]\n", rl.preSegPart(1))
	fmt.Printf("[%s]\n", rl.preSegPart(2))
	// Output:
	// [He]
	// [o, ]
	// [!]
}
