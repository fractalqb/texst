package texst

import (
	"fmt"
	"regexp"
	"testing"

	"git.fractalqb.de/fractalqb/testerr"
)

func Test_refLine(t *testing.T) {
	defer func() {
		if p := recover(); p != nil {
			fmt.Println("panic:", p)
		}
	}()
	x := RefLine{
		text: "Dec 15 22:34:38 machine systemd[1]: Starting Network Manager Script Dispatcher Service...",
	}
	testerr.Shall(x.addSeg(&segment{'M', segFix, 0, 3, ``, nil})).BeNil(t)
	testerr.Shall(x.addSeg(&segment{'D', seg1UpTo, 4, 2, `\d`, nil})).BeNil(t)
	testerr.Shall(x.addSeg(&segment{'h', segFix, 7, 2, ``, nil})).BeNil(t)
	testerr.Shall(x.addSeg(&segment{'m', segFix, 10, 2, ``, nil})).BeNil(t)
	testerr.Shall(x.addSeg(&segment{'s', segFix, 13, 2, ``, nil})).BeNil(t)
	rgxStr := x.regexp()
	fmt.Printf("`%s`\n", rgxStr)
	rgx := regexp.MustCompile(rgxStr)
	if match := rgx.FindStringSubmatch(x.text); match == nil {
		fmt.Println("No match")
	} else {
		for i, m := range match {
			if i == 0 {
				fmt.Println(m)
			} else {
				fmt.Println(i, x.segs[i-1], m)
			}
		}
	}
	// TODO checks
}
