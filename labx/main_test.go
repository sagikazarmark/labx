package labx_test

import (
	"flag"
	"os"
	"testing"
)

var root *os.Root

func TestMain(m *testing.M) {
	var rootPath string

	flag.StringVar(&rootPath, "root-path", "testdata", "run tests on a custom content root")
	flag.Parse()

	var err error
	root, err = os.OpenRoot(rootPath)
	if err != nil {
		panic(err)
	}

	exitVal := m.Run()

	os.Exit(exitVal)
}
