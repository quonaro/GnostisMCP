package simhash

import (
	"hash/fnv"
	"regexp"
	"strings"
)

var tokenRe = regexp.MustCompile(`[^a-z0-9_]+`)

// Fingerprint computes a 64-bit simhash of text.
func Fingerprint(text string) uint64 {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		h := fnv.New64a()
		h.Write([]byte(text))
		return h.Sum64()
	}

	var v [64]int

	for _, tok := range tokens {
		h := fnv.New64a()
		h.Write([]byte(tok))
		hv := h.Sum64()
		for i := 0; i < 64; i++ {
			if hv&(1<<uint(i)) != 0 {
				v[i]++
			} else {
				v[i]--
			}
		}
	}

	var fp uint64
	for i := 0; i < 64; i++ {
		if v[i] > 0 {
			fp |= 1 << uint(i)
		}
	}
	return fp
}

// Hamming returns the number of differing bits between two fingerprints.
func Hamming(a, b uint64) int {
	x := a ^ b
	count := 0
	for x != 0 {
		count++
		x &= x - 1
	}
	return count
}

// Similarity returns 1.0 for identical fingerprints, 0.0 for maximally different.
func Similarity(a, b uint64) float64 {
	return 1.0 - float64(Hamming(a, b))/64.0
}

func tokenize(text string) []string {
	lower := strings.ToLower(text)
	raw := tokenRe.Split(lower, -1)
	var tokens []string
	for _, t := range raw {
		if len(t) >= 2 {
			tokens = append(tokens, t)
		}
	}
	return tokens
}
