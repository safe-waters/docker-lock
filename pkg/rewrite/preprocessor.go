package rewrite

import (
	"errors"
	"reflect"

	"github.com/safe-waters/docker-lock/pkg/kind"
	"github.com/safe-waters/docker-lock/pkg/rewrite/preprocess"
)

type preprocessor struct {
	preprocessors []preprocess.IPreprocessor
}

// NewPreprocessor creates an IPreprocessor from IPreprocessors for different
// kinds of files. At least one preprocessor must be non nil, otherwise there
// would be no way to preprocess files.
func NewPreprocessor(
	preprocessors ...preprocess.IPreprocessor,
) (IPreprocessor, error) {
	var nonNilPreprocessors []preprocess.IPreprocessor

	for _, preprocessor := range preprocessors {
		if preprocessor != nil && !reflect.ValueOf(preprocessor).IsNil() {
			nonNilPreprocessors = append(nonNilPreprocessors, preprocessor)
		}
	}

	if len(nonNilPreprocessors) == 0 {
		return nil, errors.New("non nil 'preprocessors' must be greater than 0")
	}

	return &preprocessor{preprocessors: nonNilPreprocessors}, nil
}

// PreprocessLockfile preprocesses a Lockfile before rewriting occurs.
func (p *preprocessor) PreprocessLockfile(
	lockfile map[kind.Kind]map[string][]interface{},
) (map[kind.Kind]map[string][]interface{}, error) {
	if lockfile == nil {
		return nil, errors.New("'lockfile' cannot be nil")
	}

	for _, preprocessor := range p.preprocessors {
		var err error

		lockfile, err = preprocessor.PreprocessLockfile(lockfile)
		if err != nil {
			return nil, err
		}
	}

	return lockfile, nil
}
