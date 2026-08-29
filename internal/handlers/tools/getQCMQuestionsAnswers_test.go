package tools

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func TestGetQCMQuestionsAnswersPreservesInputOrderAfterReverseCompletion(t *testing.T) {
	queries := newQCMWorkerTestQueries(t)
	restoreShuffle := replaceQCMShuffle(t, func([]int64) {})
	defer restoreShuffle()
	builder, started, releases, completed, calls := controlledQCMBuilder([]int64{10, 20, 30})
	previousBuild := buildQuestionForQCM
	buildQuestionForQCM = func(questionID, _ int64, _ *http.Request, _ *db.Queries) (config.Question, error) {
		return builder(questionID)
	}
	defer func() { buildQuestionForQCM = previousBuild }()

	type outcome struct {
		questions []config.Question
		err       error
	}
	done := make(chan outcome, 1)
	go func() {
		questions, err := GetQCMQuestionsAnswers(1, 1, httptest.NewRequest("GET", "/", nil), queries)
		done <- outcome{questions: questions, err: err}
	}()
	waitForQCMStarts(t, started, 3)
	releaseQCMBuilds(t, releases, completed, 30, 20, 10)
	result := <-done
	if result.err != nil {
		t.Fatal(result.err)
	}
	assertQCMQuestionOrder(t, result.questions, []int64{10, 20, 30})
	assertQCMBuildCalls(t, calls, []int64{10, 20, 30})
}

func TestGetQCMQuestionsAnswersCtxPreservesChosenPermutation(t *testing.T) {
	queries := newQCMWorkerTestQueries(t)
	restoreShuffle := replaceQCMShuffle(t, func(ids []int64) { copy(ids, []int64{30, 10, 20}) })
	defer restoreShuffle()
	builder, started, releases, completed, calls := controlledQCMBuilder([]int64{10, 20, 30})
	previousBuild := buildQuestionCtxForQCM
	buildQuestionCtxForQCM = func(questionID, _ int64, _ context.Context, _ *db.Queries) (config.Question, error) {
		return builder(questionID)
	}
	defer func() { buildQuestionCtxForQCM = previousBuild }()

	type outcome struct {
		questions []config.Question
		err       error
	}
	done := make(chan outcome, 1)
	go func() {
		questions, err := GetQCMQuestionsAnswersCtx(1, 1, context.Background(), queries)
		done <- outcome{questions: questions, err: err}
	}()
	waitForQCMStarts(t, started, 3)
	releaseQCMBuilds(t, releases, completed, 20, 10, 30)
	result := <-done
	if result.err != nil {
		t.Fatal(result.err)
	}
	assertQCMQuestionOrder(t, result.questions, []int64{30, 10, 20})
	assertQCMBuildCalls(t, calls, []int64{10, 20, 30})
}

func TestBuildQCMQuestionsInOrderPropagatesErrorWithoutDuplicateBuild(t *testing.T) {
	wantErr := errors.New("build failed")
	var mu sync.Mutex
	calls := make(map[int64]int)
	questions, err := buildQCMQuestionsInOrder([]int64{10, 20, 30}, func(questionID int64) (config.Question, error) {
		mu.Lock()
		calls[questionID]++
		mu.Unlock()
		if questionID == 20 {
			return config.Question{}, wantErr
		}
		return qcmQuestionForTest(questionID), nil
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if questions != nil {
		t.Fatalf("questions = %#v, want nil on error", questions)
	}
	assertQCMBuildCalls(t, calls, []int64{10, 20, 30})
}

func TestGetQCMQuestionsAnswersCtxPropagatesCancellation(t *testing.T) {
	queries := newQCMWorkerTestQueries(t)
	restoreShuffle := replaceQCMShuffle(t, func([]int64) {})
	defer restoreShuffle()
	previousBuild := buildQuestionCtxForQCM
	buildQuestionCtxForQCM = func(_ int64, _ int64, ctx context.Context, _ *db.Queries) (config.Question, error) {
		return config.Question{}, ctx.Err()
	}
	defer func() { buildQuestionCtxForQCM = previousBuild }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	questions, err := GetQCMQuestionsAnswersCtx(1, 1, ctx, queries)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if questions != nil {
		t.Fatalf("questions = %#v, want nil", questions)
	}
}

func newQCMWorkerTestQueries(t *testing.T) *db.Queries {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE qcm(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL);
		CREATE TABLE qcm_questions(id INTEGER PRIMARY KEY,qcm_id INTEGER NOT NULL,question_id INTEGER NOT NULL,user_id INTEGER NOT NULL,position INTEGER NOT NULL);
		INSERT INTO qcm VALUES(1,1);
		INSERT INTO qcm_questions VALUES(1,1,10,1,1),(2,1,20,1,2),(3,1,30,1,3);
	`); err != nil {
		t.Fatal(err)
	}
	return db.New(conn)
}

func controlledQCMBuilder(ids []int64) (
	func(int64) (config.Question, error),
	<-chan int64,
	map[int64]chan struct{},
	<-chan int64,
	map[int64]int,
) {
	started := make(chan int64, len(ids))
	completed := make(chan int64, len(ids))
	releases := make(map[int64]chan struct{}, len(ids))
	calls := make(map[int64]int, len(ids))
	var mu sync.Mutex
	for _, id := range ids {
		releases[id] = make(chan struct{})
	}
	build := func(questionID int64) (config.Question, error) {
		mu.Lock()
		calls[questionID]++
		mu.Unlock()
		started <- questionID
		<-releases[questionID]
		completed <- questionID
		return qcmQuestionForTest(questionID), nil
	}
	return build, started, releases, completed, calls
}

func replaceQCMShuffle(t *testing.T, replacement func([]int64)) func() {
	t.Helper()
	previous := shuffleQCMQuestionIDs
	shuffleQCMQuestionIDs = replacement
	return func() { shuffleQCMQuestionIDs = previous }
}

func waitForQCMStarts(t *testing.T, started <-chan int64, count int) {
	t.Helper()
	seen := make(map[int64]bool, count)
	for range count {
		seen[<-started] = true
	}
	if len(seen) != count {
		t.Fatalf("started IDs = %v, want %d distinct builds", seen, count)
	}
}

func releaseQCMBuilds(t *testing.T, releases map[int64]chan struct{}, completed <-chan int64, order ...int64) {
	t.Helper()
	for _, id := range order {
		close(releases[id])
		if got := <-completed; got != id {
			t.Fatalf("completion = %d, want %d", got, id)
		}
	}
}

func qcmQuestionForTest(id int64) config.Question {
	return config.Question{Tags: config.Tags{MainQuestionID: id}, Content: string(rune(id))}
}

func assertQCMQuestionOrder(t *testing.T, questions []config.Question, want []int64) {
	t.Helper()
	got := make([]int64, len(questions))
	for index, question := range questions {
		got[index] = question.Tags.MainQuestionID
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("question order = %v, want %v", got, want)
	}
}

func assertQCMBuildCalls(t *testing.T, calls map[int64]int, ids []int64) {
	t.Helper()
	for _, id := range ids {
		if calls[id] != 1 {
			t.Errorf("question %d build count = %d, want 1", id, calls[id])
		}
	}
}
