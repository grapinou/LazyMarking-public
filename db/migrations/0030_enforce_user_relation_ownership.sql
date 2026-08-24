-- +goose Up
-- SQLite foreign keys ensure that an ID exists, while these triggers additionally
-- ensure that every related row belongs to the same user.

CREATE TRIGGER questions_owner_insert BEFORE INSERT ON questions
WHEN NOT EXISTS (SELECT 1 FROM subjects WHERE id = NEW.subject_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM themes WHERE id = NEW.theme_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM year_levels WHERE id = NEW.year_level_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM skills WHERE id = NEW.skill_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM difficulties WHERE id = NEW.difficulty_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM points WHERE id = NEW.point_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question resources must belong to user'); END;
CREATE TRIGGER questions_owner_update BEFORE UPDATE OF subject_id, theme_id, year_level_id, skill_id, difficulty_id, point_id, user_id ON questions
WHEN NOT EXISTS (SELECT 1 FROM subjects WHERE id = NEW.subject_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM themes WHERE id = NEW.theme_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM year_levels WHERE id = NEW.year_level_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM skills WHERE id = NEW.skill_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM difficulties WHERE id = NEW.difficulty_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM points WHERE id = NEW.point_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question resources must belong to user'); END;

CREATE TRIGGER alt_questions_owner_insert BEFORE INSERT ON alt_questions
WHEN NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question must belong to user'); END;
CREATE TRIGGER alt_questions_owner_update BEFORE UPDATE OF question_id, user_id ON alt_questions
WHEN NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question must belong to user'); END;

CREATE TRIGGER answers_owner_insert BEFORE INSERT ON answers
WHEN NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question must belong to user'); END;
CREATE TRIGGER answers_owner_update BEFORE UPDATE OF question_id, user_id ON answers
WHEN NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question must belong to user'); END;

CREATE TRIGGER images_owner_insert BEFORE INSERT ON images
WHEN NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question must belong to user'); END;
CREATE TRIGGER images_owner_update BEFORE UPDATE OF question_id, user_id ON images
WHEN NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question must belong to user'); END;

CREATE TRIGGER alt_answers_owner_insert BEFORE INSERT ON alt_answers
WHEN NOT EXISTS (SELECT 1 FROM alt_questions WHERE id = NEW.alt_question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'alternative question must belong to user'); END;
CREATE TRIGGER alt_answers_owner_update BEFORE UPDATE OF alt_question_id, user_id ON alt_answers
WHEN NOT EXISTS (SELECT 1 FROM alt_questions WHERE id = NEW.alt_question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'alternative question must belong to user'); END;

CREATE TRIGGER alt_images_owner_insert BEFORE INSERT ON alt_images
WHEN NOT EXISTS (SELECT 1 FROM alt_questions WHERE id = NEW.alt_question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'alternative question must belong to user'); END;
CREATE TRIGGER alt_images_owner_update BEFORE UPDATE OF alt_question_id, user_id ON alt_images
WHEN NOT EXISTS (SELECT 1 FROM alt_questions WHERE id = NEW.alt_question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'alternative question must belong to user'); END;

CREATE TRIGGER student_classes_owner_insert BEFORE INSERT ON student_class_codes
WHEN NOT EXISTS (SELECT 1 FROM students WHERE id = NEW.student_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM class_codes WHERE id = NEW.class_code_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'student and class must belong to user'); END;
CREATE TRIGGER student_classes_owner_update BEFORE UPDATE OF student_id, class_code_id, user_id ON student_class_codes
WHEN NOT EXISTS (SELECT 1 FROM students WHERE id = NEW.student_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM class_codes WHERE id = NEW.class_code_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'student and class must belong to user'); END;

CREATE TRIGGER qcm_questions_owner_insert BEFORE INSERT ON qcm_questions
WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id = NEW.qcm_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'QCM and question must belong to user'); END;
CREATE TRIGGER qcm_questions_owner_update BEFORE UPDATE OF qcm_id, question_id, user_id ON qcm_questions
WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id = NEW.qcm_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'QCM and question must belong to user'); END;

CREATE TRIGGER exams_owner_insert BEFORE INSERT ON exams
WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id = NEW.qcm_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM class_codes WHERE id = NEW.class_code_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM periods WHERE id = NEW.period_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM years WHERE id = NEW.year_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'exam resources must belong to user'); END;
CREATE TRIGGER exams_owner_update BEFORE UPDATE OF qcm_id, class_code_id, period_id, year_id, user_id ON exams
WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id = NEW.qcm_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM class_codes WHERE id = NEW.class_code_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM periods WHERE id = NEW.period_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM years WHERE id = NEW.year_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'exam resources must belong to user'); END;

CREATE TRIGGER generated_exams_owner_insert BEFORE INSERT ON exams_generated
WHEN NOT EXISTS (SELECT 1 FROM exams WHERE id = NEW.exam_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'exam must belong to user'); END;
CREATE TRIGGER generated_exams_owner_update BEFORE UPDATE OF exam_id, user_id ON exams_generated
WHEN NOT EXISTS (SELECT 1 FROM exams WHERE id = NEW.exam_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'exam must belong to user'); END;

CREATE TRIGGER student_exams_owner_insert BEFORE INSERT ON student_exam
WHEN NOT EXISTS (SELECT 1 FROM exams_generated WHERE id = NEW.exam_generated_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM students WHERE id = NEW.student_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'generated exam and student must belong to user'); END;
CREATE TRIGGER student_exams_owner_update BEFORE UPDATE OF exam_generated_id, student_id, user_id ON student_exam
WHEN NOT EXISTS (SELECT 1 FROM exams_generated WHERE id = NEW.exam_generated_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM students WHERE id = NEW.student_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'generated exam and student must belong to user'); END;

CREATE TRIGGER student_exam_content_owner_insert BEFORE INSERT ON student_exam_content
WHEN NOT EXISTS (SELECT 1 FROM student_exam WHERE id = NEW.student_exam_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'student exam must belong to user'); END;
CREATE TRIGGER student_exam_content_owner_update BEFORE UPDATE OF student_exam_id, user_id ON student_exam_content
WHEN NOT EXISTS (SELECT 1 FROM student_exam WHERE id = NEW.student_exam_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'student exam must belong to user'); END;

CREATE TRIGGER student_exam_pages_owner_insert BEFORE INSERT ON student_exam_page_content
WHEN NOT EXISTS (SELECT 1 FROM student_exam WHERE id = NEW.student_exam_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'student exam must belong to user'); END;
CREATE TRIGGER student_exam_pages_owner_update BEFORE UPDATE OF student_exam_id, user_id ON student_exam_page_content
WHEN NOT EXISTS (SELECT 1 FROM student_exam WHERE id = NEW.student_exam_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'student exam must belong to user'); END;

-- +goose Down
DROP TRIGGER student_exam_pages_owner_update;
DROP TRIGGER student_exam_pages_owner_insert;
DROP TRIGGER student_exam_content_owner_update;
DROP TRIGGER student_exam_content_owner_insert;
DROP TRIGGER student_exams_owner_update;
DROP TRIGGER student_exams_owner_insert;
DROP TRIGGER generated_exams_owner_update;
DROP TRIGGER generated_exams_owner_insert;
DROP TRIGGER exams_owner_update;
DROP TRIGGER exams_owner_insert;
DROP TRIGGER qcm_questions_owner_update;
DROP TRIGGER qcm_questions_owner_insert;
DROP TRIGGER student_classes_owner_update;
DROP TRIGGER student_classes_owner_insert;
DROP TRIGGER alt_images_owner_update;
DROP TRIGGER alt_images_owner_insert;
DROP TRIGGER alt_answers_owner_update;
DROP TRIGGER alt_answers_owner_insert;
DROP TRIGGER images_owner_update;
DROP TRIGGER images_owner_insert;
DROP TRIGGER answers_owner_update;
DROP TRIGGER answers_owner_insert;
DROP TRIGGER alt_questions_owner_update;
DROP TRIGGER alt_questions_owner_insert;
DROP TRIGGER questions_owner_update;
DROP TRIGGER questions_owner_insert;
