// Package preprocess provides functionality to preprocess a Lockfile
// before rewriting.
package preprocess

import "github.com/safe-waters/docker-lock/pkg/kind"

// IPreprocessor provides an interface for Preprocessors, which can modify
// a Lockfile before rewriting.
type IPreprocessor interface {
	Kind() kind.Kind
	PreprocessLockfile(
		lockfile map[kind.Kind]map[string][]interface{},
	) (map[kind.Kind]map[string][]interface{}, error)
}
