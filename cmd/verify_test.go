package cmd

import (
	"fmt"
	"path/filepath"
	"testing"
)

var verifyComposeBaseDir = filepath.Join("testdata", "verify", "compose")
var verifyDockerBaseDir = filepath.Join("testdata", "verify", "docker")

// TestVerifyComposefileDiffNumImages ensures an error occurs if
// there are a different number of images in the docker-compose file
// referenced in the lockfile.
func TestVerifyComposefileDiffNumImages(t *testing.T) {
	t.Parallel()
	outpath := filepath.Join(verifyComposeBaseDir, "diffnumimages", "docker-lock.json")
	flags := []string{fmt.Sprintf("--outpath=%s", outpath)}
	shouldFail := true
	testVerify(t, flags, shouldFail)
}

// TestVerifyComposeDiffDigests ensures an error occurs if
// a digest found in the lockfile differs from the generated
// digest from the docker-compose file.
func TestVerifyComposefileDiffDigests(t *testing.T) {
	t.Parallel()
	outpath := filepath.Join(verifyComposeBaseDir, "diffdigests", "docker-lock.json")
	flags := []string{fmt.Sprintf("--outpath=%s", outpath)}
	shouldFail := true
	testVerify(t, flags, shouldFail)
}

// TestVerifyDockerfileDiffNumImages ensures an error occurs if
// there are a different number of images in the Dockerfile
// referenced in the lockfile.
func TestVerifyDockerfileDiffNumImages(t *testing.T) {
	t.Parallel()
	outpath := filepath.Join(verifyDockerBaseDir, "diffnumimages", "docker-lock.json")
	flags := []string{fmt.Sprintf("--outpath=%s", outpath)}
	shouldFail := true
	testVerify(t, flags, shouldFail)
}

// TestVerifyDockerfileDiffDigests ensures an error occurs if
// a digest found in the lockfile differs from the generated
// digest from the Dockerfile.
func TestVerifyDockerfileDiffDigests(t *testing.T) {
	t.Parallel()
	outpath := filepath.Join(verifyDockerBaseDir, "diffdigests", "docker-lock.json")
	flags := []string{fmt.Sprintf("--outpath=%s", outpath)}
	shouldFail := true
	testVerify(t, flags, shouldFail)
}

func testVerify(t *testing.T, flags []string, shouldFail bool) {
	verifyCmd := NewVerifyCmd()
	args := append([]string{"lock", "verify"}, flags...)
	verifyCmd.SetArgs(args)
	if err := verifyCmd.Execute(); err != nil && !shouldFail {
		t.Error(err)
	}
}
