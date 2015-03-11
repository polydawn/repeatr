package output

type Output interface {
	// stub

	// See docs for input/input.go ; this is very very presumptory ATM and is liable to violent change.
	Apply(rootPath string) <-chan error
}
