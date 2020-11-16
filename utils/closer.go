package utils

import (
	"io"
	"log"
)

func CheckedClose(c io.Closer, name string) {
	err := c.Close()
	if err != nil {
		log.Fatalf("Failed to close %v: %v", name, err)
	}
}
