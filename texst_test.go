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
