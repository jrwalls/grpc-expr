package utils

import "log"

func must[T any](v T, err error) T {
	if err != nil {
		log.Fatalf("failed: %s", err)
	}

	return v
}
