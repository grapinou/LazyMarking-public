package db

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
)

func TestCreateQCMQuestionAppendsPositionsIndependently(t *testing.T) {
	conn, queries := newQCMRelationshipTestDB(t)
	ctx := context.Background()
	for _, questionID := range []int64{12, 10, 11} {
		rows, err := queries.CreateQCMQuestion(ctx, CreateQCMQuestionParams{QcmID: 3, QuestionID: questionID, UserID: 1})
		if err != nil || rows != 1 {
			t.Fatalf("create question %d rows=%d err=%v", questionID, rows, err)
		}
	}
	assertQCMPositions(t, conn, 3, [][2]int64{{12, 1}, {10, 2}, {11, 3}})

	if _, err := conn.Exec("INSERT INTO qcm VALUES (4, 'other owned', 1)"); err != nil {
		t.Fatal(err)
	}
	rows, err := queries.CreateQCMQuestion(ctx, CreateQCMQuestionParams{QcmID: 4, QuestionID: 11, UserID: 1})
	if err != nil || rows != 1 {
		t.Fatalf("create first question in other QCM rows=%d err=%v", rows, err)
	}
	assertQCMPositions(t, conn, 4, [][2]int64{{11, 1}})
}

func TestQCMQuestionReadsFollowPosition(t *testing.T) {
	conn, queries := newQCMRelationshipTestDB(t)
	if _, err := conn.Exec(`
		INSERT INTO qcm_questions(id,qcm_id,question_id,user_id,position) VALUES
			(301,3,10,1,3), (302,3,11,1,1), (303,3,12,1,2)
	`); err != nil {
		t.Fatal(err)
	}

	ids, err := queries.GetQCMQuestionsIDs(context.Background(), GetQCMQuestionsIDsParams{UserID: 1, QcmID: 3})
	if err != nil {
		t.Fatal(err)
	}
	if want := []int64{11, 12, 10}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("question IDs = %v, want %v", ids, want)
	}

	rows, err := queries.GetAllQuestionsByQCMID(context.Background(), GetAllQuestionsByQCMIDParams{UserID: 1, QcmID: 3})
	if err != nil {
		t.Fatal(err)
	}
	var got [][2]int64
	for _, row := range rows {
		got = append(got, [2]int64{row.QuestionID, row.Position})
	}
	if want := [][2]int64{{11, 1}, {12, 2}, {10, 3}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("composition = %v, want %v", got, want)
	}
}

func assertQCMPositions(t *testing.T, conn *sql.DB, qcmID int64, want [][2]int64) {
	t.Helper()
	rows, err := conn.Query("SELECT question_id,position FROM qcm_questions WHERE qcm_id=? ORDER BY position", qcmID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got [][2]int64
	for rows.Next() {
		var item [2]int64
		if err := rows.Scan(&item[0], &item[1]); err != nil {
			t.Fatal(err)
		}
		got = append(got, item)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("QCM %d positions = %v, want %v", qcmID, got, want)
	}
}
