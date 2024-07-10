package texst

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"testing"

	"git.fractalqb.de/fractalqb/testerr"
)

func ExampleRefReader() {
	ref, err := NewRefString("test text",
		`> foo bar baz
 .    xxx`)
	if err != nil {
		fmt.Println(err)
		return
	}
	rl, err := ref.NextLine()
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Println(err)
		return
	}
	fmt.Println(rl.regexp())
	fmt.Println(ref.NextLine())
	// Output:
	// ^foo (.{3}) baz$
	// <nil> test text:2:EOF
}

func TestRefReader_ILGs(t *testing.T) {
	ref := testerr.Shall1(NewRefString(t.Name(),
		`%%MDhms
> foo`,
	)).BeNil(t)
	if !slices.Equal(ref.IGroups(), []rune{'M', 'D', 'h', 'm', 's'}) {
		t.Errorf("wrong ILG: %v", ref.IGroups())
	}
	rl := testerr.Shall1(ref.NextLine()).BeNil(t)
	match := rl.match([]byte("foo"))
	if match == nil {
		t.Fatal("missing match")
	}
}
