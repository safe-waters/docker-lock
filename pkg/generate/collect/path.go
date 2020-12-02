package collect

import "github.com/safe-waters/docker-lock/pkg/kind"

type path struct {
	kind kind.Kind
	val  string
	err  error
}

// NewPath is a path of a specific kind, such as a Dockerfile,
// Composefile, or Kubernetesfile.
func NewPath(kind kind.Kind, val string, err error) IPath {
	return &path{
		kind: kind,
		val:  val,
		err:  err,
	}
}

// Kind is a getter for the kind.
func (p *path) Kind() kind.Kind {
	return p.kind
}

// SetKind is a setter for the kind.
func (p *path) SetKind(kind kind.Kind) {
	p.kind = kind
}

// Val is a getter for the name of the path.
func (p *path) Val() string {
	return p.val
}

// SetVal is a setter for the name of the path.
func (p *path) SetVal(val string) {
	p.val = val
}

// Err is a getter for the error.
func (p *path) Err() error {
	return p.err
}

// SetErr is a setter for the error.
func (p *path) SetErr(err error) {
	p.err = err
}
