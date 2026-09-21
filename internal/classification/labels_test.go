package classification

import "testing"

func TestKey(t *testing.T) {
	for _, group := range [][]string{
		{"Seconde", "seconde", " SECONDE ", "\tSeconde\n"},
		{"Électricité", "electricite", "ÉLECTRICITÉ", "E\u0301lectricite\u0301"},
		{"Physique Chimie", " Physique  Chimie ", "PHYSIQUE\tCHIMIE", "Physique\u00a0Chimie"},
		{"Physique-Chimie", "physique-chimie"},
	} {
		want := Key(group[0])
		for _, value := range group {
			if got := Key(value); got != want {
				t.Errorf("Key(%q)=%q want %q", value, got, want)
			}
			if Key(Key(value)) != Key(value) {
				t.Errorf("not idempotent: %q", value)
			}
		}
	}
	for _, pair := range [][2]string{{"2nde", "Seconde"}, {"PC", "Physique-Chimie"}, {"6ème", "Sixième"}, {"Physique-Chimie", "Physique Chimie"}, {"Seconde", "Lycée - Seconde"}} {
		if Key(pair[0]) == Key(pair[1]) {
			t.Errorf("unexpected equivalence: %q / %q", pair[0], pair[1])
		}
	}
	if Key(" \t\n\u00a0") != "" {
		t.Fatal("whitespace key not empty")
	}
}
