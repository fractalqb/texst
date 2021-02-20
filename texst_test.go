package texst

import (
	"fmt"
	"testing"
)

func Example() {
	cmpr := Compare{
		OnMismatch: func(sn int, s string, refs []*RefLine) bool {
			for _, ref := range refs {
				fmt.Printf("mismatch %d/%d: '%s' / '%s'\n",
					sn, ref.Line(),
					s, ref.Text())
			}
			return false
		},
	}
	err := cmpr.Strings(`\%12
*=ttt tt tt tt tt ttt
>1Jun 27 21:58:11.112 INFO  [thread1] create localization dir:test1/test.xCuf/l10n
 +                                                                       xxxx
>2Jun 27 21:58:11.113 INFO  [thread2] load state from file:test1/test.xCuf/bcplus.json
 +                                                                    xxxx
>1Jun 27 18:58:11.125 DEBUG [thread1] clearing maps`,
		`Jun 27 21:58:11.112 INFO  [thread1] create localization dir:test1/test.RnD/l10n
Jun 27 18:58:11.125 DEBUG [thread1] clearing MAPS
Jun 27 21:58:11.113 INFO  [thread2] load state from file:test1/test.Rnd/bcplus.json`,
	)
	fmt.Println(err)
	// Output:
	// mismatch 2/7: 'Jun 27 18:58:11.125 DEBUG [thread1] clearing MAPS' / 'Jun 27 18:58:11.125 DEBUG [thread1] clearing maps'
	// mismatch 2/5: 'Jun 27 18:58:11.125 DEBUG [thread1] clearing MAPS' / 'Jun 27 21:58:11.113 INFO  [thread2] load state from file:test1/test.xCuf/bcplus.json'
	// 1 mismatch
}

func TestCompare_good(t *testing.T) {
	noError := func(ref, subj string) func(*testing.T) {
		return func(t *testing.T) {
			cmpr := Compare{
				OnMismatch: func(n int, l string, rs []*RefLine) bool {
					t.Errorf("%3d:%s", n, l)
					for _, r := range rs {
						t.Errorf("R %c:%s", r.IGroup(), r.Text())
					}
					return false
				},
			}
			err := cmpr.Strings(ref, subj)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Run("basic", noError(`#1
> Subject might match this and must match that
 =                    xxxx`,
		"Subject might match ---- and must match that",
	))
	t.Run("basic igroups", noError(`#2
\%12
>1In group 1
>2In group 2`,
		`In group 2
In group 1`,
	))
	t.Run("unicode", noError(`#3
> Hello, 世界!
 =       xx`,
		"Hello, Go!",
	))
}

func TestCompare_miss(t *testing.T) {
	wantError := func(ref, subj string) func(*testing.T) {
		return func(t *testing.T) {
			cmpr := Compare{
				OnMismatch: func(n int, l string, rs []*RefLine) bool {
					t.Logf("%3d:%s", n, l)
					for _, r := range rs {
						t.Logf("R %c:%s", r.IGroup(), r.Text())
					}
					return false
				},
			}
			err := cmpr.Strings(ref, subj)
			if err == nil {
				t.Fatal("no missmatch detected")
			} else if _, ok := err.(MismatchCount); !ok {
				t.Fatalf("expected missmatch count but got error: %s", err)
			}
		}
	}
	t.Run("fixed too long", wantError(`#1
> head ABCD tail
 =     xxxx`,
		`head ABCD_ tail`,
	))
	t.Run("fixed too short", wantError(`#2
> head ABCD tail
 =     xxxx`,
		`head ABD tail`,
	))

	t.Run("opt nothing", wantError(`#3
> head ABCD tail
 *     xxxx`,
		`head  Tail`,
	))
	t.Run("opt too short", wantError(`#4
> head ABCD tail
 *     xxxx`,
		`head ABC Tail`,
	))
	t.Run("opt too long", wantError(`#5
> head ABCD tail
 *     xxxx`,
		`head ABCDE Tail`,
	))

	t.Run("var nothing", wantError(`#6
> head ABCD tail
 +     xxxx`,
		`head  tail`,
	))
}
