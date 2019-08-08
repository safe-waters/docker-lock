package generate

import (
	"path/filepath"
	"testing"
)

func TestCollectDockerfilesDefault(t *testing.T) {
	f, err := NewFlags([]string{})
	if err != nil {
		t.Fatal(err)
	}
	files, err := collectDockerfiles(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("Got %d Dockerfiles. Expected 0.", len(files))
	}
}

func TestCollectDockerfilesDuplicate(t *testing.T) {
	args := []string{"-f", "Dockerfile", "-f", "Dockerfile"}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	files, err := collectDockerfiles(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("Got %d Dockerfiles. Expected 1.", len(files))
	}
}

func TestCollectDockerfilesMultiple(t *testing.T) {
	baseDir := filepath.Join("testdata", "collect")
	dockerfile1 := filepath.Join(baseDir, "Dockerfile")
	dockerfile2 := filepath.Join(baseDir, "recursive", "Dockerfile")
	args := []string{"-f", dockerfile1, "-f", dockerfile2}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	expectedFiles := map[string]bool{
		dockerfile1: false,
		dockerfile2: false,
	}
	resultFiles, err := collectDockerfiles(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(resultFiles) != 2 {
		t.Fatalf("Got %d Dockerfiles. Expected 2.", len(resultFiles))
	}
	for _, resultFile := range resultFiles {
		if _, ok := expectedFiles[resultFile]; !ok {
			t.Fatalf("Got '%s'. Expected file to be a key in the map '%v'.", resultFile, expectedFiles)
		}
	}
}

func TestCollectDockerfilesRecursive(t *testing.T) {
	collectDir := filepath.Join("testdata", "collect")
	args := []string{"-r", "-rd", collectDir}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	expectedFiles := map[string]bool{
		filepath.Join(collectDir, "Dockerfile"):              false,
		filepath.Join(collectDir, "recursive", "Dockerfile"): false,
	}
	resultFiles, err := collectDockerfiles(f)
	if len(resultFiles) != len(expectedFiles) {
		t.Fatalf("Got %d files. Expected %d.", len(resultFiles), len(expectedFiles))
	}
	for _, resultFile := range resultFiles {
		if _, ok := expectedFiles[resultFile]; !ok {
			t.Fatalf("Got '%s'. Expected file to be a key in the map '%v'.", resultFile, expectedFiles)
		}
	}
}

func TestCollectDockerfilesGlobs(t *testing.T) {
	collectDir := filepath.Join("testdata", "collect")
	globPattern := filepath.Join(collectDir, "**", "Dockerfile")
	args := []string{"-g", globPattern}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	expectedFiles := map[string]bool{
		filepath.Join(collectDir, "recursive", "Dockerfile"): false,
	}
	resultFiles, err := collectDockerfiles(f)
	if len(resultFiles) != len(expectedFiles) {
		t.Fatalf("Got %d files. Expected %d.", len(resultFiles), len(expectedFiles))
	}
	for _, resultFile := range resultFiles {
		if _, ok := expectedFiles[resultFile]; !ok {
			t.Fatalf("Got '%s'. Expected file to be a key in the map '%v'.", resultFile, expectedFiles)
		}
	}
}

func TestCollectDockerfilesMultipleGlobs(t *testing.T) {
	collectDir := filepath.Join("testdata", "collect")
	globPattern1 := filepath.Join(collectDir, "**", "Dockerfile")
	globPattern2 := filepath.Join(collectDir, "Dockerfile")
	args := []string{"-g", globPattern1, "-g", globPattern2}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	expectedFiles := map[string]bool{
		filepath.Join(collectDir, "Dockerfile"):              false,
		filepath.Join(collectDir, "recursive", "Dockerfile"): false,
	}
	resultFiles, err := collectDockerfiles(f)
	if len(resultFiles) != len(expectedFiles) {
		t.Fatalf("Got %d files. Expected %d.", len(resultFiles), len(expectedFiles))
	}
	for _, resultFile := range resultFiles {
		if _, ok := expectedFiles[resultFile]; !ok {
			t.Fatalf("Got '%s'. Expected file to be a key in the map '%v'.", resultFile, expectedFiles)
		}
	}
}

func TestCollectComposefilesDefault(t *testing.T) {
	f, err := NewFlags([]string{})
	if err != nil {
		t.Fatal(err)
	}
	files, err := collectComposefiles(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("Got %d Composefiles. Expected 0.", len(files))
	}
}

func TestCollectComposefilesDuplicate(t *testing.T) {
	args := []string{"-cf", "docker-compose.yml", "-cf", "docker-compose.yml"}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	files, err := collectComposefiles(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("Got %d Composefiles. Expected 1.", len(files))
	}
}

func TestCollectComposefilesMultiple(t *testing.T) {
	baseDir := filepath.Join("testdata", "collect")
	composefile1 := filepath.Join(baseDir, "docker-compose.yml")
	composefile2 := filepath.Join(baseDir, "recursive", "docker-compose.yml")
	args := []string{"-cf", composefile1, "-cf", composefile2}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	expectedFiles := map[string]bool{
		composefile1: false,
		composefile2: false,
	}
	resultFiles, err := collectComposefiles(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(resultFiles) != 2 {
		t.Fatalf("Got %d Composefiles. Expected 2.", len(resultFiles))
	}
	for _, resultFile := range resultFiles {
		if _, ok := expectedFiles[resultFile]; !ok {
			t.Fatalf("Got '%s'. Expected file to be a key in the map '%v'.", resultFile, expectedFiles)
		}
	}
}

func TestCollectComposefilesRecursive(t *testing.T) {
	collectDir := filepath.Join("testdata", "collect")
	args := []string{"-cr", "-crd", collectDir}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	expectedFiles := map[string]bool{
		filepath.Join(collectDir, "docker-compose.yml"):               false,
		filepath.Join(collectDir, "recursive", "docker-compose.yaml"): false,
	}
	resultFiles, err := collectComposefiles(f)
	if len(resultFiles) != len(expectedFiles) {
		t.Fatalf("Got %d files. Expected %d.", len(resultFiles), len(expectedFiles))
	}
	for _, resultFile := range resultFiles {
		if _, ok := expectedFiles[resultFile]; !ok {
			t.Fatalf("Got '%s'. Expected file to be a key in the map '%v'.", resultFile, expectedFiles)
		}
	}
}

func TestCollectComposefileGlobs(t *testing.T) {
	collectDir := filepath.Join("testdata", "collect")
	globPattern := filepath.Join(collectDir, "**", "*yaml")
	args := []string{"-cg", globPattern}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	expectedFiles := map[string]bool{
		filepath.Join(collectDir, "recursive", "docker-compose.yaml"): false,
	}
	resultFiles, err := collectComposefiles(f)
	if len(resultFiles) != len(expectedFiles) {
		t.Fatalf("Got %d files. Expected %d.", len(resultFiles), len(expectedFiles))
	}
	for _, resultFile := range resultFiles {
		if _, ok := expectedFiles[resultFile]; !ok {
			t.Fatalf("Got '%s'. Expected file to be a key in the map '%v'.", resultFile, expectedFiles)
		}
	}
}

func TestCollectComposefilesMultipleGlobs(t *testing.T) {
	collectDir := filepath.Join("testdata", "collect")
	globPattern1 := filepath.Join(collectDir, "**", "*.yaml")
	globPattern2 := filepath.Join(collectDir, "*.yml")
	args := []string{"-cg", globPattern1, "-cg", globPattern2}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	expectedFiles := map[string]bool{
		filepath.Join(collectDir, "docker-compose.yml"):               false,
		filepath.Join(collectDir, "recursive", "docker-compose.yaml"): false,
	}
	resultFiles, err := collectComposefiles(f)
	if len(resultFiles) != len(expectedFiles) {
		t.Fatalf("Got %d files. Expected %d.", len(resultFiles), len(expectedFiles))
	}
	for _, resultFile := range resultFiles {
		if _, ok := expectedFiles[resultFile]; !ok {
			t.Fatalf("Got '%s'. Expected file to be a key in the map '%v'.", resultFile, expectedFiles)
		}
	}
}
