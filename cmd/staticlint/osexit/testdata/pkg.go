package pkg1

import "os"

func mulfunc(i int) (int, error) {
	return i * 2, nil
}

func errCheckFunc() {
	os.Exit(1) // want "os exit usage prohibited"
}
