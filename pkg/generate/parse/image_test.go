package parse_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

func TestImage(t *testing.T) {
	t.Parallel()

	image := parse.NewImage(kind.Dockerfile, "", "", "", nil, nil)

	image.SetName("busybox")

	if image.Name() != "busybox" {
		t.Fatal(
			fmt.Errorf("expected name %s, got %s", "busybox", image.Name()),
		)
	}

	image.SetTag("latest")

	if image.Tag() != "latest" {
		t.Fatal(
			fmt.Errorf("expected tag %s, got %s", "latest", image.Tag()),
		)
	}

	image.SetDigest(testutils.BusyboxLatestSHA)

	if image.Digest() != testutils.BusyboxLatestSHA {
		t.Fatal(
			fmt.Errorf(
				"expected digest %s, got %s",
				testutils.BusyboxLatestSHA, image.Digest(),
			),
		)
	}

	image.SetKind(kind.Composefile)

	if image.Kind() != kind.Composefile {
		t.Fatal(
			fmt.Errorf(
				"expected kind %s, got %s", kind.Composefile, image.Digest(),
			),
		)
	}

	metadata := map[string]interface{}{
		"position": 0,
	}

	image.SetMetadata(metadata)

	metadata["position"] = 1

	if metadata = image.Metadata(); metadata["position"] != 0 {
		t.Fatal(
			errors.New("unexpected ability to modify metadata after setting"),
		)
	}

	metadata = image.Metadata()

	metadata["position"] = 1

	if metadata = image.Metadata(); metadata["position"] != 0 {
		t.Fatal(
			errors.New("unexpected ability to modify metadata after getting"),
		)
	}

	imageLine := fmt.Sprintf(
		"busybox:latest@sha256:%s", testutils.BusyboxLatestSHA,
	)

	if image.ImageLine() != imageLine {
		t.Fatal(
			fmt.Errorf(
				"expected image line %s, got %s", imageLine, image.ImageLine(),
			),
		)
	}

	imageLine = fmt.Sprintf("golang:1.14@sha256:%s", testutils.GolangLatestSHA)
	image.SetNameTagDigestFromImageLine(imageLine)

	if image.ImageLine() != imageLine {
		t.Fatal(
			fmt.Errorf(
				"expected image line %s, got %s", imageLine, image.ImageLine(),
			),
		)
	}

	if image.Name() != "golang" {
		t.Fatal(
			fmt.Errorf("expected name %s, got %s", "golang", image.Name()),
		)
	}

	if image.Tag() != "1.14" {
		t.Fatal(
			fmt.Errorf("expected tag %s, got %s", "1.14", image.Tag()),
		)
	}

	if image.Digest() != testutils.GolangLatestSHA {
		t.Fatal(
			fmt.Errorf(
				"expected digest %s, got %s",
				testutils.GolangLatestSHA, image.Digest(),
			),
		)
	}
}
