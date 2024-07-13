package texst

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"git.fractalqb.de/fractalqb/testerr"
)

func ExampleTexst() {
	ref, err := NewRefString("test text", `> foo bar baz
 .    xxx`)
	if err != nil {
		fmt.Println(err)
		return
	}
	vrf := Texst{
		OnMismatch: func(tiLNo int, tiLine []byte, _ []*RefLine) {
			fmt.Printf("input:%d [%s]\n", tiLNo, tiLine)
		},
		OnMatch: func(tiLNo int, line []byte, _ *RefLine, match []int) {
			txt := func(i int) string {
				i *= 2
				part := line[match[i]:match[i+1]]
				return string(part)
			}
			fmt.Printf("match:%d [%s]:", tiLNo, txt(0))
			for i := range (len(match) / 2) - 1 {
				fmt.Printf(" %d=[%s]", i, txt(i+1))
			}
			fmt.Println()
		},
	}
	mismatchCount, err := vrf.Check(ref, strings.NewReader(
		`foo bar baz`,
	))
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%d mismatches\n", mismatchCount)
	}
	// Output:
	// match:1 [foo bar baz]: 0=[bar]
	// 0 mismatches
}

func TestTexst_maskTypes(t *testing.T) {
	check := func(t *testing.T, ref, subj string, mm int) {
		refRd := testerr.Shall1(NewRefString(t.Name(), ref)).BeNil(t)
		mmn := testerr.Shall1((&Texst{}).Check(refRd, strings.NewReader(subj))).BeNil(t)
		if mmn != mm {
			t.Errorf("expect %d, detected %d mismatches [%s]", mm, mmn, subj)
		}
	}

	t.Run("fix", func(t *testing.T) {
		const ref = `> foo bar baz
 .    xxx`
		check(t, ref, "foo XXX baz", 0)
		check(t, ref, "foo XX baz", 2)
		check(t, ref, "foo XXXX baz", 2)
	})
	t.Run("0 or more", func(t *testing.T) {
		const ref = `> foo bar baz
 *    xxx`
		check(t, ref, "foo  baz", 0)
		check(t, ref, "foo X baz", 0)
		check(t, ref, "foo XXX baz", 0)
		check(t, ref, "foo XXXX baz", 0)
	})
	t.Run("1 or more", func(t *testing.T) {
		const ref = `> foo bar baz
 +    xxx`
		check(t, ref, "foo  baz", 2)
		check(t, ref, "foo X baz", 0)
		check(t, ref, "foo XXX baz", 0)
		check(t, ref, "foo XXXX baz", 0)
	})
	t.Run("0 up to mask", func(t *testing.T) {
		const ref = `> foo bar baz
 0    xxx`
		check(t, ref, "foo  baz", 0)
		check(t, ref, "foo X baz", 0)
		check(t, ref, "foo XXX baz", 0)
		check(t, ref, "foo XXXX baz", 2)
	})

	t.Run("1 up to mask", func(t *testing.T) {
		const ref = `> foo bar baz
 1    xxx`
		check(t, ref, "foo  baz", 2)
		check(t, ref, "foo X baz", 0)
		check(t, ref, "foo XXX baz", 0)
		check(t, ref, "foo XXXX baz", 2)
	})
	t.Run("at least mask", func(t *testing.T) {
		const ref = `> foo bar baz
 -    xxx`
		check(t, ref, "foo  baz", 2)
		check(t, ref, "foo XX baz", 2)
		check(t, ref, "foo XXX baz", 0)
		check(t, ref, "foo XXXX baz", 0)
	})
	t.Run("char class", func(t *testing.T) {
		const ref = `> foo bar baz
 .    xxx
 ?x \d`
		check(t, ref, "foo abc baz", 2)
		check(t, ref, "foo 123 baz", 0)
		check(t, ref, "foo 1_3 baz", 2)
	})
	t.Run("match", func(t *testing.T) {
		const ref = `> foo bar baz
 .    xxx
 ~x \d{3}`
		check(t, ref, "foo 12 baz", 2)
		check(t, ref, "foo 123 baz", 0)
		check(t, ref, "foo 1_3 baz", 2)
		check(t, ref, "foo 1234 baz", 2)
	})
}

func TestTexst_inputLen(t *testing.T) {
	check := func(subj string) (mmLines []string) {
		refRd := testerr.Shall1(NewRefString(t.Name(),
			`> line 1
> line 2
> line 3`)).BeNil(t)
		txs := Texst{OnMismatch: func(testedNo int, testedLine []byte, ref []*RefLine) {
			mmLines = append(mmLines, fmt.Sprintf("%d %s %d",
				testedNo,
				testedLine,
				len(ref),
			))
		}}
		testerr.Shall1(txs.Check(refRd, strings.NewReader(subj))).BeNil(t)
		return mmLines
	}
	t.Run("subject match", func(t *testing.T) {
		mmls := check("line 1\nline 2\nline 3")
		if len(mmls) != 0 {
			t.Error("mismatches", mmls)
		}
	})
	t.Run("subject too long", func(t *testing.T) {
		mmls := check("line 1\nline 2\nline 3\nline 4")
		if !slices.Equal(mmls, []string{"4 line 4 0"}) {
			t.Error("mismatches", mmls)
		}
	})
	t.Run("subject too short", func(t *testing.T) {
		mmls := check("line 1\nline 2\n")
		if !slices.Equal(mmls, []string{"3  1"}) {
			t.Error("mismatches", mmls)
		}
	})
}

func TestTexst_iGroups(t *testing.T) {
	refRd := testerr.Shall1(NewRefString(t.Name(),
		`%%12
>1line 1
>1line 3
>2line 2
>2line 4`)).BeNil(t)
	mmn := testerr.Shall1((&Texst{}).Check(refRd, strings.NewReader(
		`line 1
line 2
line 3
line 4`,
	))).BeNil(t)
	if mmn != 0 {
		t.Error("unexpected mismatch")
	}
}
