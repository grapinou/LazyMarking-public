package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestQuestionVariantIdentitySnapshotEvolutionIsBackwardCompatible(t *testing.T) {
	current, err := json.Marshal(QCM{Questions: []Question{{Tags: Tags{
		MainQuestionID: 42,
		VariantType:    AltQuestion,
		VariantID:      7,
	}}}})
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(current)
	if !strings.Contains(encoded, `"main_question_id":42`) || !strings.Contains(encoded, `"variant_type":"altQuestion"`) || !strings.Contains(encoded, `"variant_id":7`) {
		t.Fatalf("new snapshot lacks stable identities: %s", encoded)
	}

	const legacy = `{"questions":[{"tags":{"main_question_id":42,"point":{"name":2}},"content":"Snapshot historique"}]}`
	var historical QCM
	if err := json.Unmarshal([]byte(legacy), &historical); err != nil {
		t.Fatal(err)
	}
	if len(historical.Questions) != 1 || historical.Questions[0].Tags.MainQuestionID != 42 || historical.Questions[0].Tags.VariantType != "" || historical.Questions[0].Tags.VariantID != 0 {
		t.Fatalf("legacy snapshot changed semantics: %+v", historical)
	}
	roundTrip, err := json.Marshal(historical)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(roundTrip), "variant_type") || strings.Contains(string(roundTrip), "variant_id") {
		t.Fatalf("legacy snapshot gained invented identity: %s", roundTrip)
	}
}
