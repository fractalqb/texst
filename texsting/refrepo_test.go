package texsting

import (
	"net/http"
	"strings"
	"testing"
)

func TestFatal_Example(t *testing.T) {
	const subject = `Jun 29 20:58:11.112 INFO  [thread1] create localization dir:test1/test.xCuf/l10n
Jun 29 20:58:11.113 INFO  [thread2] load state from file:test1/test.xCuf/bcplus.json
Jun 29 20:58:11.125 DEBUG [thread1] clearing maps`
	// Used to create initial reference: Record(t, "", strings.NewReader(subject))
	// Now here comes the test:
	Fatal(t, "", strings.NewReader(subject))
}

func TestError(t *testing.T) {
	if !testing.Verbose() {
		t.Skip("skip test with remote calls")
	}
	resp, err := http.Get("https://httpbin.org/get")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	Error(t, "", resp.Body)
}
