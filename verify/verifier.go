package verify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/michaelperel/docker-lock/generate"
	"github.com/michaelperel/docker-lock/registry"
	"github.com/spf13/cobra"
)

type Verifier struct {
	*generate.Generator
	*generate.Lockfile
	outPath string
}

func NewVerifier(cmd *cobra.Command) (*Verifier, error) {
	outPath, err := cmd.Flags().GetString("outpath")
	if err != nil {
		return nil, err
	}
	lByt, err := ioutil.ReadFile(outPath)
	if err != nil {
		return nil, err
	}
	var lFile generate.Lockfile
	if err := json.Unmarshal(lByt, &lFile); err != nil {
		return nil, err
	}
	var i int
	cFpaths := make([]string, len(lFile.ComposefileImages))
	for fpath := range lFile.ComposefileImages {
		cFpaths[i] = filepath.FromSlash(fpath)
		i++
	}
	i = 0
	dFpaths := make([]string, len(lFile.DockerfileImages))
	for fpath := range lFile.DockerfileImages {
		dFpaths[i] = filepath.FromSlash(fpath)
		i++
	}
	dockerfileEnvBuildArgs, err := cmd.Flags().GetBool("dockerfile-env-build-args")
	if err != nil {
		return nil, err
	}
	g := &generate.Generator{Dockerfiles: dFpaths, Composefiles: cFpaths, DockerfileEnvBuildArgs: dockerfileEnvBuildArgs}
	return &Verifier{Generator: g, Lockfile: &lFile, outPath: outPath}, nil
}

func (v *Verifier) VerifyLockfile(wrapperManager *registry.WrapperManager) error {
	lByt, err := v.GenerateLockfileBytes(wrapperManager)
	if err != nil {
		return err
	}
	var lFile generate.Lockfile
	if err := json.Unmarshal(lByt, &lFile); err != nil {
		return err
	}
	err = errors.New("Failed to verify.")
	if len(v.DockerfileImages) != len(lFile.DockerfileImages) {
		err = fmt.Errorf("%s Got %d Dockerfiles. Want %d.",
			err,
			len(lFile.DockerfileImages),
			len(v.DockerfileImages))
		return err
	}
	if len(v.ComposefileImages) != len(lFile.ComposefileImages) {
		err = fmt.Errorf("%s Got %d Composefiles. Want %d.",
			err,
			len(lFile.ComposefileImages),
			len(v.ComposefileImages))
		return err
	}
	var dImagesWG sync.WaitGroup
	dImagesErrs := make(chan error)
	for dFpath := range v.DockerfileImages {
		dImagesWG.Add(1)
		go func(dFpath string) {
			defer dImagesWG.Done()
			if len(v.DockerfileImages[dFpath]) != len(lFile.DockerfileImages[dFpath]) {
				err = fmt.Errorf("%s Got %d images in file %s. Want %d.",
					err,
					len(lFile.DockerfileImages[dFpath]),
					dFpath,
					len(v.DockerfileImages[dFpath]))
				dImagesErrs <- err
				return
			}
			for i := range v.DockerfileImages[dFpath] {
				if v.DockerfileImages[dFpath][i] != lFile.DockerfileImages[dFpath][i] {
					err = fmt.Errorf("%s Got image:\n%+v\nWant image:\n%+v",
						err,
						lFile.DockerfileImages[dFpath][i].Prettify(),
						v.DockerfileImages[dFpath][i].Prettify())
					dImagesErrs <- err
					return
				}
			}
		}(dFpath)
	}
	go func() {
		dImagesWG.Wait()
		close(dImagesErrs)
	}()
	for err := range dImagesErrs {
		return err
	}
	var cImagesWG sync.WaitGroup
	cImagesErrs := make(chan error)
	for cFpath := range v.ComposefileImages {
		cImagesWG.Add(1)
		go func(cFpath string) {
			defer cImagesWG.Done()
			if len(v.ComposefileImages[cFpath]) != len(lFile.ComposefileImages[cFpath]) {
				err = fmt.Errorf("%s Got %d images in file %s. Want %d.",
					err,
					len(lFile.ComposefileImages[cFpath]),
					cFpath,
					len(v.ComposefileImages[cFpath]))
				cImagesErrs <- err
				return
			}
			for i := range v.ComposefileImages[cFpath] {
				if v.ComposefileImages[cFpath][i] != lFile.ComposefileImages[cFpath][i] {
					err = fmt.Errorf("%s Got image:\n%+v\nWant image:\n%+v",
						err,
						lFile.ComposefileImages[cFpath][i].Prettify(),
						v.ComposefileImages[cFpath][i].Prettify())
					cImagesErrs <- err
					return
				}
			}
		}(cFpath)
	}
	go func() {
		cImagesWG.Wait()
		close(cImagesErrs)
	}()
	for err := range cImagesErrs {
		return err
	}
	return nil
}
