package write

type writtenPath struct {
	originalPath string
	newPath      string
	err          error
}

// NewWrittenPath returns an IWrittenPath that contains information
// linking a newly written file and its original.
func NewWrittenPath(
	originalPath string,
	newPath string,
	err error,
) IWrittenPath {
	return &writtenPath{
		originalPath: originalPath,
		newPath:      newPath,
		err:          err,
	}
}

// OriginalPath is a getter for the original path.
func (w *writtenPath) OriginalPath() string {
	return w.originalPath
}

// SetOriginalPath is a setter for the original path.
func (w *writtenPath) SetOriginalPath(originalPath string) {
	w.originalPath = originalPath
}

// NewPath is a getter for the new path.
func (w *writtenPath) NewPath() string {
	return w.newPath
}

// SetNewPath is a setter for the new path.
func (w *writtenPath) SetNewPath(newPath string) {
	w.newPath = newPath
}

// Err is a getter for the error.
func (w *writtenPath) Err() error {
	return w.err
}

// SetErr is a setter for the error.
func (w *writtenPath) SetErr(err error) {
	w.err = err
}
