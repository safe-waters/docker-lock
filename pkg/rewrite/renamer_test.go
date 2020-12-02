package rewrite_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/rewrite"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

func TestRenamer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name           string
		RewrittenPaths []write.IWrittenPath
		Expected       [][]byte
	}{
		{
			Name: "Rename Rewritten Path to Original",
			RewrittenPaths: []write.IWrittenPath{
				write.NewWrittenPath("Dockerfile1", "TempDockerfile1", nil),
				write.NewWrittenPath("Dockerfile2", "TempDockerfile2", nil),
			},
			Expected: [][]byte{[]byte("temporary1"), []byte("temporary2")},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := testutils.MakeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			var (
				paths            []string
				originalPaths    []string
				originalContents [][]byte
				renamer          = rewrite.NewRenamer()
				rewrittenPathsCh = make(
					chan write.IWrittenPath, len(test.RewrittenPaths),
				)
			)

			for _, rewrittenPath := range test.RewrittenPaths {
				originalPaths = append(
					originalPaths, rewrittenPath.OriginalPath(),
				)
				originalContents = append(originalContents, []byte("original"))

				paths = append(paths, rewrittenPath.NewPath())

				rewrittenPath.SetOriginalPath(
					filepath.Join(tempDir, rewrittenPath.OriginalPath()),
				)

				rewrittenPath.SetNewPath(
					filepath.Join(tempDir, rewrittenPath.NewPath()),
				)

				rewrittenPathsCh <- rewrittenPath
			}

			close(rewrittenPathsCh)

			testutils.WriteFilesToTempDir(
				t, tempDir, originalPaths, originalContents,
			)
			testutils.WriteFilesToTempDir(t, tempDir, paths, test.Expected)

			if err := renamer.RenameFiles(rewrittenPathsCh); err != nil {
				t.Fatal(err)
			}

			var got []string

			for _, rewrittenPath := range test.RewrittenPaths {
				got = append(got, rewrittenPath.OriginalPath())
			}

			testutils.AssertWrittenFilesEqual(t, test.Expected, got)
		})
	}
}
