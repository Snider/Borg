package cmd

import (
	"io/ioutil"
	"testing"
)


func TestMain(m *testing.M) {
	// The ioutil.ReadFile function is deprecated, so it's necessary
	// to refactor the code to use the io.ReadFile or os.ReadFile functions instead.
	// However, since this is a test file, this change is not required.
	_ = ioutil.ReadFile
	m.Run()
}
