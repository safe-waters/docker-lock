package generate

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/format"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
	"github.com/safe-waters/docker-lock/pkg/generate/registry/contrib"
	"github.com/safe-waters/docker-lock/pkg/generate/registry/firstparty"
	"github.com/safe-waters/docker-lock/pkg/generate/update"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

// DefaultPathCollector creates an IPathCollector that works with Dockerfiles,
// Composefiles, and Kubernetesfiles.
//
// For all three, respectively, the defaults are
// ["Dockerfile"], ["docker-compose.yml", "docker-compose.yaml"], and
// ["deployment.yml", "deployment.yaml", "pod.yml", "pod.yaml",
// "job.yml", "job.yaml"].
//
// PathCollectors are set according to the flag, "ExcludePaths".
// If all "ExcludePaths" are true or any of the three's flags,
// are nil, an error is returned.
func DefaultPathCollector(flags *Flags) (generate.IPathCollector, error) {
	if err := ensureFlagsNotNil(flags); err != nil {
		return nil, err
	}

	if flags.DockerfileFlags.ExcludePaths &&
		flags.ComposefileFlags.ExcludePaths &&
		flags.KubernetesfileFlags.ExcludePaths {
		return nil, errors.New("nothing to do - all paths excluded")
	}

	var (
		dockerfileCollector     collect.IPathCollector
		composefileCollector    collect.IPathCollector
		kubernetesfileCollector collect.IPathCollector
		err                     error
	)

	if !flags.DockerfileFlags.ExcludePaths {
		dockerfileCollector, err = collect.NewPathCollector(
			kind.Dockerfile,
			flags.FlagsWithSharedValues.BaseDir, []string{"Dockerfile"},
			flags.DockerfileFlags.ManualPaths, flags.DockerfileFlags.Globs,
			flags.DockerfileFlags.Recursive,
		)
		if err != nil {
			return nil, err
		}
	}

	if !flags.ComposefileFlags.ExcludePaths {
		composefileCollector, err = collect.NewPathCollector(
			kind.Composefile,
			flags.FlagsWithSharedValues.BaseDir,
			[]string{"docker-compose.yml", "docker-compose.yaml"},
			flags.ComposefileFlags.ManualPaths, flags.ComposefileFlags.Globs,
			flags.ComposefileFlags.Recursive,
		)
		if err != nil {
			return nil, err
		}
	}

	if !flags.KubernetesfileFlags.ExcludePaths {
		kubernetesfileCollector, err = collect.NewPathCollector(
			kind.Kubernetesfile,
			flags.FlagsWithSharedValues.BaseDir,
			[]string{
				"deployment.yml", "deployment.yaml",
				"pod.yml", "pod.yaml",
				"job.yml", "job.yaml",
			},
			flags.KubernetesfileFlags.ManualPaths,
			flags.KubernetesfileFlags.Globs,
			flags.KubernetesfileFlags.Recursive,
		)
		if err != nil {
			return nil, err
		}
	}

	return generate.NewPathCollector(
		dockerfileCollector, composefileCollector, kubernetesfileCollector,
	)
}

// DefaultImageParser creates an IImageParser that works with Dockerfiles,
// Composefiles, and Kubernetesfiles.
//
// ImageParsers are set according to the flag, "ExcludePaths".
// If all "ExcludePaths" are true or any of the three's flags,
// are nil, an error is returned.
func DefaultImageParser(flags *Flags) (generate.IImageParser, error) {
	if err := ensureFlagsNotNil(flags); err != nil {
		return nil, err
	}

	if flags.DockerfileFlags.ExcludePaths &&
		flags.ComposefileFlags.ExcludePaths &&
		flags.KubernetesfileFlags.ExcludePaths {
		return nil, errors.New("nothing to do - all paths excluded")
	}

	var (
		dockerfileImageParser     parse.IDockerfileImageParser
		composefileImageParser    parse.IComposefileImageParser
		kubernetesfileImageParser parse.IKubernetesfileImageParser
	)

	if !flags.DockerfileFlags.ExcludePaths ||
		!flags.ComposefileFlags.ExcludePaths {
		dockerfileImageParser = parse.NewDockerfileImageParser()
	}

	if !flags.ComposefileFlags.ExcludePaths {
		var err error

		composefileImageParser, err = parse.NewComposefileImageParser(
			dockerfileImageParser,
		)

		if err != nil {
			return nil, err
		}
	}

	if !flags.KubernetesfileFlags.ExcludePaths {
		kubernetesfileImageParser = parse.NewKubernetesfileImageParser()
	}

	return generate.NewImageParser(
		dockerfileImageParser, composefileImageParser,
		kubernetesfileImageParser,
	)
}

// DefaultImageFormatter creates an IImageFormatter that works with
// Dockerfiles, Composefiles, and Kubernetesfiles.
//
// ImageFormatters are set according to the flag, "ExcludePaths".
// If all "ExcludePaths" are true or any of the three's flags,
// are nil, an error is returned.
func DefaultImageFormatter(flags *Flags) (generate.IImageFormatter, error) {
	if err := ensureFlagsNotNil(flags); err != nil {
		return nil, err
	}

	if flags.DockerfileFlags.ExcludePaths &&
		flags.ComposefileFlags.ExcludePaths &&
		flags.KubernetesfileFlags.ExcludePaths {
		return nil, errors.New("nothing to do - all paths excluded")
	}

	dockerfileImageFormatter := format.NewDockerfileImageFormatter()
	composefileImageFormatter := format.NewComposefileImageFormatter()
	kubernetesfileImageFormatter := format.NewKubernetesfileImageFormatter()

	return generate.NewImageFormatter(
		dockerfileImageFormatter, composefileImageFormatter,
		kubernetesfileImageFormatter,
	)
}

// DefaultImageDigestUpdater creates an IImageDigestUpdater that works with
// Dockerfiles, Composefiles, and Kubernetesfiles.
//
// If all "ExcludePaths" are true or any of the three's flags,
// are nil, an error is returned.
//
// DefaultImageDigestUpdater relies on DefaultWrapperManager and is subject
// to the same error conditions.
func DefaultImageDigestUpdater(
	client *registry.HTTPClient,
	flags *Flags,
) (generate.IImageDigestUpdater, error) {
	if err := ensureFlagsNotNil(flags); err != nil {
		return nil, err
	}

	if flags.DockerfileFlags.ExcludePaths &&
		flags.ComposefileFlags.ExcludePaths &&
		flags.KubernetesfileFlags.ExcludePaths {
		return nil, errors.New("nothing to do - all paths excluded")
	}

	wrapperManager, err := DefaultWrapperManager(
		client, flags.FlagsWithSharedValues.ConfigPath,
	)
	if err != nil {
		return nil, err
	}

	imageDigestUpdater, err := update.NewImageDigestUpdater(
		wrapperManager, flags.FlagsWithSharedValues.IgnoreMissingDigests,
	)
	if err != nil {
		return nil, err
	}

	return generate.NewImageDigestUpdater(imageDigestUpdater)
}

// DefaultConfigPath returns the default location of docker's config.json
// for all platforms. If the platform does not have a home directory, it
// returns an empty string.
func DefaultConfigPath() string {
	if homeDir, err := os.UserHomeDir(); err == nil {
		configPath := filepath.Join(homeDir, ".docker", "config.json")
		if _, err := os.Stat(configPath); err != nil {
			return ""
		}

		return configPath
	}

	return ""
}

// DefaultWrapperManager creates a WrapperManager for querying registries
// for image digests. The returned wrapper manager uses all possible first party
// and contrib wrappers. The default wrapper queries Dockerhub for digests
// and is used if the manager is unable to select a more specific wrapper.
func DefaultWrapperManager(
	client *registry.HTTPClient,
	configPath string,
) (*registry.WrapperManager, error) {
	defaultWrapper, err := firstparty.DefaultWrapper(client, configPath)
	if err != nil {
		return nil, err
	}

	wrapperManager := registry.NewWrapperManager(defaultWrapper)
	wrapperManager.Add(firstparty.AllWrappers(client, configPath)...)
	wrapperManager.Add(contrib.AllWrappers(client, configPath)...)

	return wrapperManager, nil
}

// DefaultLoadEnv loads .env files based on the path. If a path does not
// exist and that path is not ".env", an error will occur.
func DefaultLoadEnv(path string) error {
	if _, err := os.Stat(path); err != nil {
		if path == ".env" {
			return nil
		}

		return err
	}

	return godotenv.Load(path)
}

func ensureFlagsNotNil(flags *Flags) error {
	if flags == nil {
		return errors.New("flags cannot be nil")
	}

	if flags.DockerfileFlags == nil {
		return errors.New("flags.DockerfileFlags cannot be nil")
	}

	if flags.ComposefileFlags == nil {
		return errors.New("flags.ComposefileFlags cannot be nil")
	}

	if flags.KubernetesfileFlags == nil {
		return errors.New("flags.KubernetesfileFlags cannot be nil")
	}

	if flags.FlagsWithSharedValues == nil {
		return errors.New("flags.FlagsWithSharedValues cannot be nil")
	}

	return nil
}
