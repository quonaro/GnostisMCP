package simhash

import "testing"

func TestFingerprint_IdenticalText(t *testing.T) {
	a := Fingerprint("func main() { fmt.Println('hello') }")
	b := Fingerprint("func main() { fmt.Println('hello') }")
	if a != b {
		t.Errorf("identical text should produce identical fingerprints: %d vs %d", a, b)
	}
	if Similarity(a, b) != 1.0 {
		t.Errorf("identical fingerprints should have similarity 1.0, got %f", Similarity(a, b))
	}
}

func TestFingerprint_OneLineChange(t *testing.T) {
	original := `func process(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}
	result := transform(data)
	return store(result)
}`

	modified := `func process(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("data is empty")
	}
	result := transform(data)
	return store(result)
}`

	fpA := Fingerprint(original)
	fpB := Fingerprint(modified)
	sim := Similarity(fpA, fpB)
	if sim < 0.85 {
		t.Errorf("one-line change in 7-line function should have similarity >= 0.85, got %.4f", sim)
	}
}

func TestFingerprint_UnrelatedText(t *testing.T) {
	a := Fingerprint("func main() { fmt.Println('hello world') }")
	b := Fingerprint("class DatabaseConnection: def __init__(self, host, port):")
	sim := Similarity(a, b)
	if sim > 0.6 {
		t.Errorf("unrelated texts should have similarity < 0.6, got %.4f", sim)
	}
}

func TestFingerprint_EmptyString(t *testing.T) {
	fp := Fingerprint("")
	if fp == 0 {
		t.Error("empty string should not produce zero fingerprint")
	}
	fp2 := Fingerprint("")
	if fp != fp2 {
		t.Error("empty string should produce deterministic fingerprint")
	}
}

func TestHamming(t *testing.T) {
	if Hamming(0, 0) != 0 {
		t.Error("Hamming(0,0) should be 0")
	}
	if Hamming(0, 0xFFFFFFFFFFFFFFFF) != 64 {
		t.Error("Hamming of all-different should be 64")
	}
	if Hamming(1, 3) != 1 {
		t.Errorf("Hamming(1,3) should be 1, got %d", Hamming(1, 3))
	}
}

func TestSimilarity(t *testing.T) {
	if Similarity(0, 0) != 1.0 {
		t.Error("Similarity(0,0) should be 1.0")
	}
	if Similarity(0, 0xFFFFFFFFFFFFFFFF) != 0.0 {
		t.Error("Similarity of all-different should be 0.0")
	}
}
