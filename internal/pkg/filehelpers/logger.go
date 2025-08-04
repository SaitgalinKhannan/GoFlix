package filehelpers

import (
	"log"
	"os"
)

func CloseFile(f *os.File) {
	err := f.Close()
	if err != nil {
		log.Printf("[filehelpers] error closing %s: %v\n", f.Name(), err)
	}
}

func OsRemove(path string) {
	err := os.Remove(path)
	if err != nil {
		log.Printf("[filehelpers] error removing %s: %v\n", path, err)
	}
}
