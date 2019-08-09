package verify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/michaelperel/docker-lock/generate"
	"github.com/michaelperel/docker-lock/registry"
)

type Verifier struct {
	*generate.Generator
	outfile string
}

func NewVerifier(flags *Flags) (*Verifier, error) {
	lockfileByt, err := ioutil.ReadFile(flags.Outfile)
	if err != nil {
		return nil, err
	}
	var lockfile generate.Lockfile
	if err := json.Unmarshal(lockfileByt, &lockfile); err != nil {
		return nil, err
	}
	for i := range lockfile.Generator.Dockerfiles {
		lockfile.Generator.Dockerfiles[i] = filepath.FromSlash(lockfile.Generator.Dockerfiles[i])
	}
	for i := range lockfile.Generator.Composefiles {
		lockfile.Generator.Composefiles[i] = filepath.FromSlash(lockfile.Generator.Composefiles[i])
	}
	return &Verifier{Generator: lockfile.Generator, outfile: flags.Outfile}, nil
}

func (v *Verifier) VerifyLockfile(wrapperManager *registry.WrapperManager) error {
	lockfileBytes, err := ioutil.ReadFile(v.outfile)
	if err != nil {
		return err
	}
	verificationBytes, err := v.GenerateLockfileBytes(wrapperManager)
	if err != nil {
		return err
	}
	var existingLockfile, verificationLockfile generate.Lockfile
	if err := json.Unmarshal(lockfileBytes, &existingLockfile); err != nil {
		return err
	}
	if err := json.Unmarshal(verificationBytes, &verificationLockfile); err != nil {
		return err
	}
	errMsg := errors.New("Failed to verify.")
	if len(existingLockfile.Images) != len(verificationLockfile.Images) {
		errMsg = fmt.Errorf("%s Found %d files. Expected %d files.",
			errMsg,
			len(verificationLockfile.Images),
			len(existingLockfile.Images))
		return errMsg
	}
	for fileName := range existingLockfile.Images {
		if len(existingLockfile.Images[fileName]) != len(verificationLockfile.Images[fileName]) {
			errMsg = fmt.Errorf("%s Found %d images in file %s. Expected %d files.",
				errMsg,
				len(verificationLockfile.Images[fileName]),
				fileName,
				len(existingLockfile.Images[fileName]))
			return errMsg
		}
		for i := range existingLockfile.Images[fileName] {
			if existingLockfile.Images[fileName][i] != verificationLockfile.Images[fileName][i] {
				errMsg = fmt.Errorf("%s Found image:\n%+v\nExpected image:\n%+v",
					errMsg,
					verificationLockfile.Images[fileName][i],
					existingLockfile.Images[fileName][i])
				return errMsg
			}
		}
	}
	return nil
}
