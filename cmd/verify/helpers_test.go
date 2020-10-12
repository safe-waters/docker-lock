package verify_test

import (
	"encoding/json"
	"testing"

	"github.com/safe-waters/docker-lock/cmd/verify"
)

func assertFlagsEqual(
	t *testing.T,
	expected *verify.Flags,
	got *verify.Flags,
) {
	t.Helper()

	if *expected != *got {
		t.Fatalf(
			"expected %+v, got %+v",
			jsonPrettyPrint(t, expected), jsonPrettyPrint(t, got),
		)
	}
}

func jsonPrettyPrint(t *testing.T, i interface{}) string {
	t.Helper()

	byt, err := json.MarshalIndent(i, "", "\t")
	if err != nil {
		t.Fatal(err)
	}

	return string(byt)
}
