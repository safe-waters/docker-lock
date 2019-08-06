package verify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/michaelperel/docker-lock/generate"
	"github.com/michaelperel/docker-lock/registry"
	"io/ioutil"
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
	if bytes.Equal(lockfileBytes, verificationBytes) {
		return nil
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
		errMsg = fmt.Errorf("%s Found %d images. Expected %d images.", errMsg, len(verificationLockfile.Images), len(existingLockfile.Images))
		return errMsg
	}
	for i, _ := range existingLockfile.Images {
		if existingLockfile.Images[i] != verificationLockfile.Images[i] {
			errMsg = fmt.Errorf("%s Found image:\n%+v\nExpected image:\n%+v\n", errMsg, verificationLockfile.Images[i], existingLockfile.Images[i])
			return errMsg
		}
	}
	return errMsg
}
