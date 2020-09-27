package rewrite_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/rewrite"
	"github.com/safe-waters/docker-lock/pkg/rewrite/writers"
)

func TestRenamer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name           string
		RewrittenPaths []*writers.WrittenPath
		Expected       [][]byte
	}{
		{
			Name: "Rename Rewritten Path to Original",
			RewrittenPaths: []*writers.WrittenPath{
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

			tempDir := generateUUID(t)
			makeDir(t, tempDir)
			defer os.RemoveAll(tempDir)

			renamer := &rewrite.Renamer{}

			rewrittenPathsCh := make(
				chan *writers.WrittenPath, len(test.RewrittenPaths),
			)

			for i, rewrittenPath := range test.RewrittenPaths {
				rewrittenPath.OriginalPath = filepath.Join(
					tempDir, rewrittenPath.OriginalPath,
				)
				rewrittenPath.Path = filepath.Join(tempDir, rewrittenPath.Path)
				writeFile(
					t, rewrittenPath.OriginalPath, []byte("original"),
				)
				writeFile(
					t, rewrittenPath.Path, test.Expected[i],
				)

				rewrittenPathsCh <- rewrittenPath
			}

			close(rewrittenPathsCh)

			if err := renamer.RenameFiles(rewrittenPathsCh); err != nil {
				t.Fatal(err)
			}

			got := make([][]byte, len(test.RewrittenPaths))

			for i, rewrittenPath := range test.RewrittenPaths {
				origByt, err := ioutil.ReadFile(rewrittenPath.OriginalPath)
				if err != nil {
					t.Fatal(err)
				}
				got[i] = origByt
			}

			assertOriginalContentsEqualPathContents(t, test.Expected, got)
		})
	}
}
