package questionfamilies

import (
	"reflect"
	"testing"
)

func TestBuildPreservesMainQuestionOrderAndSortsVariants(t *testing.T) {
	questions := []Question{
		{ID: 20, Content: "second"},
		{ID: 10, Content: "first"},
		{ID: 30, Content: "without variants"},
	}
	variants := []Variant{
		{ID: 202, QuestionID: 20, Content: "second B"},
		{ID: 102, QuestionID: 10, Content: "first B"},
		{ID: 201, QuestionID: 20, Content: "second A"},
		{ID: 101, QuestionID: 10, Content: "first A"},
	}

	got := Build(questions, variants)
	if len(got) != 3 {
		t.Fatalf("families count = %d, want 3", len(got))
	}
	if ids := []int64{got[0].Main.ID, got[1].Main.ID, got[2].Main.ID}; !reflect.DeepEqual(ids, []int64{20, 10, 30}) {
		t.Fatalf("main question order = %v, want input order [20 10 30]", ids)
	}
	if ids := variantIDs(got[0]); !reflect.DeepEqual(ids, []int64{201, 202}) {
		t.Fatalf("first family variants = %v, want [201 202]", ids)
	}
	if ids := variantIDs(got[1]); !reflect.DeepEqual(ids, []int64{101, 102}) {
		t.Fatalf("second family variants = %v, want [101 102]", ids)
	}
	if got[2].Variants == nil || len(got[2].Variants) != 0 {
		t.Fatalf("family without variants = %#v, want non-nil empty slice", got[2].Variants)
	}
}

func TestBuildIgnoresVariantWithoutAProvidedMainQuestion(t *testing.T) {
	got := Build(
		[]Question{{ID: 10, Content: "owned"}},
		[]Variant{{ID: 100, QuestionID: 99, Content: "unrelated"}},
	)
	if len(got) != 1 || len(got[0].Variants) != 0 {
		t.Fatalf("families = %#v, want one empty family", got)
	}
}

func variantIDs(family QuestionFamily) []int64 {
	ids := make([]int64, 0, len(family.Variants))
	for _, variant := range family.Variants {
		ids = append(ids, variant.ID)
	}
	return ids
}
