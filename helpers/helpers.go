package helpers

import (
	"math/rand"
)

// ID generate an ID
func ID() string {
	letter := []rune("abcdefghijklmnopqrstuvwxyz1234567890")

	b := make([]rune, 20)

	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}

	return string(b)
}
