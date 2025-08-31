package main

import (
	"github.com/grapinou/LazyMarking/internal/workflow"
)

func main() {
	baseURL := "http://localhost:8080"

	workflow.GetRegisterWF(baseURL)
	workflow.PostRegisterWF("Sighto", "aa.com", "aa", baseURL)
	workflow.LoginWf(baseURL)
	workflow.SubjectsWf(baseURL)
	workflow.ThemesWf(baseURL)
	workflow.YearLevelsWf(baseURL)
	workflow.SkillsWf(baseURL)
	workflow.DifficultiesWf(baseURL)
	workflow.PointsWf(baseURL)
	workflow.QuestionsWf(baseURL)
	workflow.AnswerWf(baseURL)
	workflow.AltQuestionWf(baseURL)
	workflow.AltAnswerWf(baseURL)
	workflow.ImageWf(baseURL)
	workflow.ClassCodesWf(baseURL)
	workflow.StudentsWf(baseURL)
	workflow.QcmWf(baseURL)
	workflow.QcmQuestionWf(baseURL)
	workflow.YearExamWf(baseURL)
	workflow.PeriodWf(baseURL)
	workflow.ExamWf(baseURL)
	workflow.GenerateExamWf(baseURL)
}
