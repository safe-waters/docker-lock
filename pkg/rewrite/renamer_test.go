package rewrite_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/rewrite"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

func TestRenamer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name           string
		RewrittenPaths []*write.WrittenPath
		Expected       [][]byte
	}{
		{
			Name: "Rename Rewritten Path to Original",
			RewrittenPaths: []*write.WrittenPath{
				{
					OriginalPath: "Dockerfile1",
					Path:         "TempDockerfile1",
				},
				{
					OriginalPath: "Dockerfile2",
					Path:         "TempDockerfile2",
				},
			},
			Expected: [][]byte{[]byte("temporary1"), []byte("temporary2")},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := makeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			renamer := &rewrite.Renamer{}

			var paths []string

			var originalPaths []string

			var originalContents [][]byte

			rewrittenPathsCh := make(
				chan *write.WrittenPath, len(test.RewrittenPaths),
			)
			for _, rewrittenPath := range test.RewrittenPaths {
				originalPaths = append(
					originalPaths, rewrittenPath.OriginalPath,
				)
				originalContents = append(originalContents, []byte("original"))

				paths = append(paths, rewrittenPath.Path)

				rewrittenPath.OriginalPath = filepath.Join(
					tempDir, rewrittenPath.OriginalPath,
				)
				rewrittenPath.Path = filepath.Join(tempDir, rewrittenPath.Path)

				rewrittenPathsCh <- rewrittenPath
			}

			close(rewrittenPathsCh)

			writeFilesToTempDir(t, tempDir, originalPaths, originalContents)
			writeFilesToTempDir(t, tempDir, paths, test.Expected)

			if err := renamer.RenameFiles(rewrittenPathsCh); err != nil {
				t.Fatal(err)
			}

			var got []string

			for _, rewrittenPath := range test.RewrittenPaths {
				got = append(got, rewrittenPath.OriginalPath)
			}

			assertWrittenFiles(t, test.Expected, got)
		})
	}
}
