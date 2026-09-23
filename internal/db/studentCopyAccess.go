package db

import (
	"context"
	"database/sql"
	"time"
)

type StudentCopyAccess struct {
	StudentExamID  int64
	GenerationID   int64
	UserID         int64
	Token          string
	CodeHash       string
	State          string
	PublishedAt    sql.NullTime
	ExpiresAt      sql.NullTime
	FailedAttempts int64
	LockedUntil    sql.NullTime
	FirstName      string
	LastName       string
	ExamName       string
	ClassName      string
	Username       string
}

const studentCopyAccessSelect = `SELECT se.id, se.exam_generated_id, se.user_id, a.public_token, a.code_hash,
 a.state, a.published_at, a.expires_at, a.failed_attempts, a.locked_until,
 s.first_name, s.last_name, e.name, cc.name, u.username
 FROM student_copy_access a
 JOIN student_exam se ON se.id=a.student_exam_id
 JOIN students s ON s.id=se.student_id AND s.user_id=se.user_id
 JOIN exams_generated eg ON eg.id=se.exam_generated_id AND eg.user_id=se.user_id
 JOIN exams e ON e.id=eg.exam_id AND e.user_id=se.user_id
 JOIN class_codes cc ON cc.id=e.class_code_id AND cc.user_id=se.user_id
 JOIN users u ON u.id=se.user_id `

func scanStudentCopyAccess(row *sql.Row) (StudentCopyAccess, error) {
	var a StudentCopyAccess
	err := row.Scan(&a.StudentExamID, &a.GenerationID, &a.UserID, &a.Token, &a.CodeHash,
		&a.State, &a.PublishedAt, &a.ExpiresAt, &a.FailedAttempts, &a.LockedUntil,
		&a.FirstName, &a.LastName, &a.ExamName, &a.ClassName, &a.Username)
	return a, err
}

func (q *Queries) GetStudentCopyAccessByToken(ctx context.Context, token string) (StudentCopyAccess, error) {
	return scanStudentCopyAccess(q.db.QueryRowContext(ctx, studentCopyAccessSelect+` WHERE a.public_token=? AND eg.status='success'`, token))
}

func (q *Queries) ListStudentCopyAccess(ctx context.Context, userID, generationID int64) ([]StudentCopyAccess, error) {
	rows, err := q.db.QueryContext(ctx, studentCopyAccessSelect+` WHERE se.user_id=? AND se.exam_generated_id=? AND eg.status='success' ORDER BY se.id`, userID, generationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []StudentCopyAccess
	for rows.Next() {
		var a StudentCopyAccess
		if err := rows.Scan(&a.StudentExamID, &a.GenerationID, &a.UserID, &a.Token, &a.CodeHash,
			&a.State, &a.PublishedAt, &a.ExpiresAt, &a.FailedAttempts, &a.LockedUntil,
			&a.FirstName, &a.LastName, &a.ExamName, &a.ClassName, &a.Username); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (q *Queries) ListStudentExamsWithoutAccess(ctx context.Context, userID, generationID int64) ([]int64, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT se.id FROM student_exam se
 JOIN exams_generated eg ON eg.id=se.exam_generated_id AND eg.user_id=se.user_id
 LEFT JOIN student_copy_access a ON a.student_exam_id=se.id
 WHERE se.user_id=? AND se.exam_generated_id=? AND eg.status='success' AND a.student_exam_id IS NULL ORDER BY se.id`, userID, generationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (q *Queries) InsertStudentCopyAccess(ctx context.Context, userID, generationID, studentExamID int64, token, hash string) error {
	result, err := q.db.ExecContext(ctx, `INSERT INTO student_copy_access(student_exam_id,public_token,code_hash)
 SELECT se.id,?,? FROM student_exam se JOIN exams_generated eg ON eg.id=se.exam_generated_id AND eg.user_id=se.user_id
 WHERE se.id=? AND se.user_id=? AND se.exam_generated_id=? AND eg.status='success'
 ON CONFLICT(student_exam_id) DO NOTHING`, token, hash, studentExamID, userID, generationID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		err := q.db.QueryRowContext(ctx, `SELECT student_exam_id FROM student_copy_access WHERE student_exam_id=?`, studentExamID).Scan(new(int64))
		return err
	}
	return nil
}

func (q *Queries) UseStudentCopyCodeAttempt(ctx context.Context, studentExamID int64, now time.Time) (bool, error) {
	result, err := q.db.ExecContext(ctx, `UPDATE student_copy_access
 SET failed_attempts=CASE WHEN locked_until IS NOT NULL AND CAST(strftime('%s',locked_until) AS INTEGER)<=? THEN 1 ELSE failed_attempts+1 END,
 locked_until=CASE WHEN locked_until IS NOT NULL AND CAST(strftime('%s',locked_until) AS INTEGER)<=? THEN NULL
   WHEN failed_attempts+1>=5 THEN ? ELSE NULL END
 WHERE student_exam_id=? AND (locked_until IS NULL OR CAST(strftime('%s',locked_until) AS INTEGER)<=?)`, now.Unix(), now.Unix(), now.Add(15*time.Minute), studentExamID, now.Unix())
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (q *Queries) ClearStudentCopyCodeFailures(ctx context.Context, studentExamID int64) error {
	_, err := q.db.ExecContext(ctx, `UPDATE student_copy_access SET failed_attempts=0,locked_until=NULL WHERE student_exam_id=?`, studentExamID)
	return err
}

func (q *Queries) PublishStudentCopyAccess(ctx context.Context, userID, generationID, studentExamID int64, now time.Time) (int64, error) {
	result, err := q.db.ExecContext(ctx, `UPDATE student_copy_access SET state='published',published_at=?,expires_at=?
 WHERE student_exam_id=? AND state='unpublished' AND EXISTS (
 SELECT 1 FROM student_exam se JOIN exams_generated eg ON eg.id=se.exam_generated_id AND eg.user_id=se.user_id
 WHERE se.id=student_copy_access.student_exam_id AND se.user_id=? AND se.exam_generated_id=? AND eg.status='success')`, now, now.Add(14*24*time.Hour), studentExamID, userID, generationID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (q *Queries) ChangeStudentCopyAccess(ctx context.Context, userID, generationID, studentExamID int64, state string, now time.Time) (int64, error) {
	var result sql.Result
	var err error
	if state == "revoked" {
		result, err = q.db.ExecContext(ctx, `UPDATE student_copy_access SET state='revoked' WHERE student_exam_id=? AND state IN ('unpublished','published') AND EXISTS (
 SELECT 1 FROM student_exam se WHERE se.id=student_copy_access.student_exam_id AND se.user_id=? AND se.exam_generated_id=?)`, studentExamID, userID, generationID)
	} else if state == "published" {
		result, err = q.db.ExecContext(ctx, `UPDATE student_copy_access SET state='published',published_at=?,expires_at=? WHERE student_exam_id=? AND (state='revoked' OR (state='published' AND CAST(strftime('%s',expires_at) AS INTEGER)<=?)) AND EXISTS (
 SELECT 1 FROM student_exam se WHERE se.id=student_copy_access.student_exam_id AND se.user_id=? AND se.exam_generated_id=?)`, now, now.Add(14*24*time.Hour), studentExamID, now.Unix(), userID, generationID)
	} else {
		return 0, sql.ErrNoRows
	}
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
