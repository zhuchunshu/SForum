//go:build !unix

package extensions

import "os"

func openStableReadFile(name string) (*os.File, error) {
	return os.Open(name)
}

func openStableRootReadFile(root *os.Root, name string) (*os.File, error) {
	return root.Open(name)
}
