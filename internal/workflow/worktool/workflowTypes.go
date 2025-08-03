package worktool

type AnswerStructWf struct {
	QuestionID string
	State      string
	Content    string
}

type AltQuestionStructWf struct {
	QuestionID string
	Content    string
}

type AltAnswerStructWf struct {
	QuestionID    string
	AltQuestionID string
	State         string
	Content       string
}
