package tools

import (
	"reflect"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestGroupQrCodesSortsExamsAndPages(t *testing.T) {
	input := []config.QrCodeInfo{
		{StudentExamID: 20, PageExam: 2, PageName: "b.png"},
		{StudentExamID: 10, PageExam: 3, PageName: "c.png"},
		{StudentExamID: 20, PageExam: 1, PageName: "a.png"},
		{StudentExamID: 10, PageExam: 1, PageName: "a.png"},
	}
	want := []config.Exam{
		{StudentExamID: 10, Pages: []config.Page{{Number: 1, Name: "a.png"}, {Number: 3, Name: "c.png"}}},
		{StudentExamID: 20, Pages: []config.Page{{Number: 1, Name: "a.png"}, {Number: 2, Name: "b.png"}}},
	}
	if got := GroupQrCodes(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("GroupQrCodes() = %#v, want %#v", got, want)
	}
}
