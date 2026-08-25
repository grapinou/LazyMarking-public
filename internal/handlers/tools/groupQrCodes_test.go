package tools

import (
	"reflect"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestGroupQrCodesSortsExamsAndPages(t *testing.T) {
	input := []config.QrCodeInfo{
		{StudentExamID: 14, PageExam: 2, PageName: "14-2.png"},
		{StudentExamID: 387, PageExam: 3, PageName: "387-3.png"},
		{StudentExamID: 14, PageExam: 1, PageName: "14-1.png"},
		{StudentExamID: 387, PageExam: 1, PageName: "387-1.png"},
		{StudentExamID: 387, PageExam: 2, PageName: "387-2.png"},
	}
	want := []config.Exam{
		{StudentExamID: 14, Pages: []config.Page{{Number: 1, Name: "14-1.png"}, {Number: 2, Name: "14-2.png"}}},
		{StudentExamID: 387, Pages: []config.Page{{Number: 1, Name: "387-1.png"}, {Number: 2, Name: "387-2.png"}, {Number: 3, Name: "387-3.png"}}},
	}
	if got := GroupQrCodes(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("GroupQrCodes() = %#v, want %#v", got, want)
	}
}
