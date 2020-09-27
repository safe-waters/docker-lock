package generate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	cmd_generate "github.com/safe-waters/docker-lock/cmd/generate"
	"github.com/safe-waters/docker-lock/pkg/generate"
)

func assertPathCollector(
	t *testing.T,
	flags *cmd_generate.Flags,
	shouldFail bool,
) {
	t.Helper()

	pathCollector, err := cmd_generate.DefaultPathCollector(flags)
	if shouldFail {
		if err == nil {
			t.Fatal("expected error but did not get one")
		}

		return
	}

	if err != nil {
		t.Fatal(err)
	}

	concretePathCollector, ok := pathCollector.(*generate.PathCollector)
	if !ok {
		t.Fatal("unexpected path collector type")
	}

	if (flags.DockerfileFlags.ExcludePaths &&
		!reflect.ValueOf(concretePathCollector.DockerfileCollector).IsNil()) ||
		(flags.ComposefileFlags.ExcludePaths &&
			!reflect.ValueOf(
				concretePathCollector.ComposefileCollector,
			).IsNil()) {
		t.Fatal("expected nil collector")
	}

	if (!flags.DockerfileFlags.ExcludePaths &&
		reflect.ValueOf(concretePathCollector.DockerfileCollector).IsNil()) ||
		(!flags.ComposefileFlags.ExcludePaths &&
			reflect.ValueOf(
				concretePathCollector.ComposefileCollector,
			).IsNil()) {
		t.Fatal("expected non nil collector")
	}
}

func assertImageParser(
	t *testing.T,
	flags *cmd_generate.Flags,
	shouldFail bool,
) {
	t.Helper()

	imageParser, err := cmd_generate.DefaultImageParser(flags)
	if shouldFail {
		if err == nil {
			t.Fatal("expected error but did not get one")
		}

		return
	}

	if err != nil {
		t.Fatal(err)
	}

	concreteImageParser, ok := imageParser.(*generate.ImageParser)
	if !ok {
		t.Fatal("unexpected image parser type")
	}

	if flags.ComposefileFlags.ExcludePaths &&
		!reflect.ValueOf(concreteImageParser.ComposefileImageParser).IsNil() {
		t.Fatal("expected nil composefile image parser")
	}

	if !flags.ComposefileFlags.ExcludePaths &&
		(reflect.ValueOf(concreteImageParser.DockerfileImageParser).IsNil() ||
			reflect.ValueOf(
				concreteImageParser.ComposefileImageParser,
			).IsNil()) {
		t.Fatal("expected non nil parsers")
	}

	if !flags.DockerfileFlags.ExcludePaths &&
		reflect.ValueOf(concreteImageParser.DockerfileImageParser).IsNil() {
		t.Fatal("expected non nil dockerfile parser")
	}

	if (flags.DockerfileFlags.ExcludePaths &&
		flags.ComposefileFlags.ExcludePaths) &&
		(!reflect.ValueOf(concreteImageParser.DockerfileImageParser).IsNil() ||
			!reflect.ValueOf(
				concreteImageParser.ComposefileImageParser,
			).IsNil()) {
		t.Fatal("expected nil parsers")
	}
}

func assertImageDigestUpdater(
	t *testing.T,
	flags *cmd_generate.Flags,
	shouldFail bool,
) {
	t.Helper()

	updater, err := cmd_generate.DefaultImageDigestUpdater(nil, flags)
	if shouldFail {
		if err == nil {
			t.Fatal("expected error but did not get one")
		}

		return
	}

	if err != nil {
		t.Fatal(err)
	}

	_, ok := updater.(*generate.ImageDigestUpdater)
	if !ok {
		t.Fatal("unexpected updater type")
	}
}

func assertFlagsEqual(
	t *testing.T,
	expected interface{},
	got interface{},
) {
	t.Helper()

	if !reflect.DeepEqual(expected, got) {
		t.Fatalf(
			"expected %+v, got %+v",
			jsonPrettyPrint(t, expected), jsonPrettyPrint(t, got),
		)
	}
}

func getAbsPath(t *testing.T) string {
	t.Helper()

	absPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		t.Fatal(err)
	}

	return absPath
}

func jsonPrettyPrint(t *testing.T, i interface{}) string {
	t.Helper()

	byt, err := json.MarshalIndent(i, "", "\t")
	if err != nil {
		t.Fatal(err)
	}

	return string(byt)
}
