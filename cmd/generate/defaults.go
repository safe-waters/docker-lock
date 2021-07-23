package generate

import (
	"errors"

	"github.com/compose-spec/compose-go/cli"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/collect"
	"github.com/safe-waters/docker-lock/pkg/generate/format"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/update"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

// DefaultPathCollector creates an IPathCollector that works with Dockerfiles,
// Composefiles, and Kubernetesfiles.
//
// For all three, respectively, the defaults are
// ["Dockerfile"], ["compose.yml", "compose.yaml",
// "docker-compose.yml", "docker-compose.yaml"], and
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
			cli.DefaultFileNames,
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

	var (
		dockerfileImageFormatter     = format.NewDockerfileImageFormatter()
		composefileImageFormatter    = format.NewComposefileImageFormatter()
		kubernetesfileImageFormatter = format.NewKubernetesfileImageFormatter()
	)

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
func DefaultImageDigestUpdater(
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

	digestRequester := update.NewDigestRequester()

	imageDigestUpdater, err := update.NewImageDigestUpdater(
		digestRequester, flags.FlagsWithSharedValues.IgnoreMissingDigests,
		flags.FlagsWithSharedValues.UpdateExistingDigests,
	)
	if err != nil {
		return nil, err
	}

	return generate.NewImageDigestUpdater(imageDigestUpdater)
}

func ensureFlagsNotNil(flags *Flags) error {
	if flags == nil {
		return errors.New("'flags' cannot be nil")
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
