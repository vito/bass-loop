package present

type Workaround struct{}

// workaround for https://github.com/livebud/bud/issues/137
func New() *Workaround {
	return &Workaround{}
}
