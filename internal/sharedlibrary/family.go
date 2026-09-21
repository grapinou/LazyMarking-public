// Package sharedlibrary implements read-only publication and independent copies
// of question families. Existing owner-scoped editing queries are unchanged.
package sharedlibrary

import (
	"context"
	"database/sql"
	"errors"
	"sort"

	"github.com/grapinou/LazyMarking/internal/db"
)

type Answer struct {
	Content string
	State   int64
}

type Image struct {
	Name   string
	Resize int64
}

type Version struct {
	// ID is zero for the main question, otherwise the variant ID.
	ID      int64
	Content string
	Answers []Answer
	Image   *Image
}

type Family struct {
	Metadata db.GetSharedQuestionRow
	Versions []Version
}

// LoadShared must run inside the caller's transaction to read a coherent family.
// Author ownership is obtained from the shared resource, never from request data.
func LoadShared(ctx context.Context, q *db.Queries, id int64) (Family, error) {
	meta, err := q.GetSharedQuestion(ctx, id)
	if err != nil {
		return Family{}, err
	}
	family := Family{Metadata: meta}
	main := Version{Content: meta.Content}
	answers, err := q.GetAllAnswersByQuestionID(ctx, db.GetAllAnswersByQuestionIDParams{QuestionID: id, UserID: meta.UserID})
	if err != nil {
		return Family{}, err
	}
	sort.Slice(answers, func(i, j int) bool { return answers[i].ID < answers[j].ID })
	for _, a := range answers {
		main.Answers = append(main.Answers, Answer{a.Content, a.State})
	}
	img, err := q.GetImageByQuestionID(ctx, db.GetImageByQuestionIDParams{QuestionID: id, UserID: meta.UserID})
	if err == nil {
		main.Image = &Image{img.ImageName, img.ResizePercentage}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Family{}, err
	}
	family.Versions = append(family.Versions, main)
	variants, err := q.GetAllAltQuestions(ctx, db.GetAllAltQuestionsParams{QuestionID: id, UserID: meta.UserID})
	if err != nil {
		return Family{}, err
	}
	sort.Slice(variants, func(i, j int) bool { return variants[i].ID < variants[j].ID })
	for _, v := range variants {
		version := Version{ID: v.ID, Content: v.Content}
		answers, err := q.GetAllAltAnswersByAltQuestionID(ctx, db.GetAllAltAnswersByAltQuestionIDParams{AltQuestionID: v.ID, UserID: meta.UserID})
		if err != nil {
			return Family{}, err
		}
		sort.Slice(answers, func(i, j int) bool { return answers[i].ID < answers[j].ID })
		for _, a := range answers {
			version.Answers = append(version.Answers, Answer{a.Content, a.State})
		}
		img, err := q.GetAltImageByAltQuestionID(ctx, db.GetAltImageByAltQuestionIDParams{AltQuestionID: v.ID, QuestionID: id, UserID: meta.UserID})
		if err == nil {
			version.Image = &Image{img.ImageName, img.ResizePercentage}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return Family{}, err
		}
		family.Versions = append(family.Versions, version)
	}
	return family, nil
}
