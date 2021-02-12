package texsting

import (
	"strings"
	"testing"
)

func TestRefRepo(t *testing.T) {
	const subject = `Jun 29 20:58:11.112 INFO  [thread1] create localization dir:test1/test.xCuf/l10n
Jun 29 20:58:11.113 INFO  [thread2] load state from file:test1/test.xCuf/bcplus.json
Jun 29 20:58:11.125 DEBUG [thread1] clearing maps`
	repo := RefRepo{Dir: "."}
	// Used to create initial reference: repo.TestRecord(t, "", strings.NewReader(subject))
	//
	// now here comes the test:
	repo.TestFatal(t, "", strings.NewReader(subject))
}
