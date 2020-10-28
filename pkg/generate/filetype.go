package generate

type FileType int

const (
	Dockerfile FileType = iota
	Composefile
)

func (f FileType) String() string {
	return [...]string{"Dockerfile", "Composefile"}[f]
}
