package texst

import (
	"fmt"
	"strings"
)

func ExampleTexst() {
	ref, err := NewRefString("test text", `> foo bar baz
 .    xxx`)
	if err != nil {
		fmt.Println(err)
		return
	}
	mismatchCount := 0
	vrf := Texst{
		OnMismatch: func(tiLNo int, tiLine []byte, _ []*RefLine) error {
			mismatchCount++
			fmt.Printf("input:%d [%s]\n", tiLNo, tiLine)
			return nil
		},
		OnMatch: func(tiLNo int, line []byte, _ *RefLine, match []int) error {
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
			return nil
		},
	}
	err = vrf.Check(ref, strings.NewReader(
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
