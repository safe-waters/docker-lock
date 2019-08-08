package generate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	args := []string{}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Dockerfiles) != 0 {
		t.Fatalf("Got %d dockerfiles. Expected 0.", len(f.Dockerfiles))
	}
	if len(f.Composefiles) != 0 {
		t.Fatalf("Got %d composefiles. Expected 0.", len(f.Composefiles))
	}
	if len(f.Globs) != 0 {
		t.Fatalf("Got %d globs. Expected 0.", len(f.Globs))
	}
	if len(f.ComposeGlobs) != 0 {
		t.Fatalf("Got %d compose globs. Expected 0.", len(f.ComposeGlobs))
	}
	if f.Recursive {
		t.Fatal("Got true for recursive. Expected false.")
	}
	if f.RecursiveDir != "." {
		t.Fatalf("Got '%s'. Expected '.'.", f.RecursiveDir)
	}
	if f.ComposeRecursive {
		t.Fatal("Got true for compose recursive. Expected false.")
	}
	if f.ComposeRecursiveDir != "." {
		t.Fatalf("Got '%s'. Expected '.'.", f.ComposeRecursiveDir)
	}
	if f.Outfile != "docker-lock.json" {
		t.Fatalf("Got '%s' outfile. Expected 'docker-lock.json'.", f.Outfile)
	}
	if f.EnvFile != ".env" {
		t.Fatalf("Got '%s' env file. Expected .env.", f.EnvFile)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	defaultConfig := filepath.Join(homeDir, ".docker", "config.json")
	if f.ConfigFile != defaultConfig {
		t.Fatalf("Got '%s' config file. Expected '%s'.", f.ConfigFile, defaultConfig)
	}
}

func TestOutFile(t *testing.T) {
	outFile := "docker-lock-test.json"
	args := []string{"-o", outFile}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	if f.Outfile != outFile {
		t.Errorf("Got '%s'. Expected '%s'", f.Outfile, outFile)
	}
}

func TestConfigFile(t *testing.T) {
	configFile := filepath.Join("testdata", "flags", "config.json")
	args := []string{"-c", configFile}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	if f.ConfigFile != configFile {
		t.Errorf("Got '%s'. Expected '%s'.", f.ConfigFile, configFile)
	}
}

func TestEnvFile(t *testing.T) {
	envFile := filepath.Join("testdata", "flags", ".env")
	args := []string{"-e", envFile}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	if f.EnvFile != envFile {
		t.Errorf("Got '%s'. Expected '%s'.", f.EnvFile, envFile)
	}
}

func TestFaultyEnvFile(t *testing.T) {
	envFile := filepath.Join("testdata", "flags", ".env2")
	args := []string{"-e", envFile}
	_, err := NewFlags(args)
	if err == nil {
		t.Fatal("Faulty env file should fail.")
	}
}
