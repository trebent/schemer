package schemer

import "testing"

func checkErr(err error, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
