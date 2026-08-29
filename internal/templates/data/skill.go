package data

import (
	"net/url"
	"strconv"
)

type SkillRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type SkillContext struct {
	ID   int64
	Name string
}

type SkillListItem struct {
	ID        int64
	Name      string
	EditURL   string
	DeleteURL string
}

func SkillURL(base string, skillID int64) string {
	return base + "?skill_id=" + url.QueryEscape(strconv.FormatInt(skillID, 10))
}

var DefaultSkillRoutes = SkillRoutes{
	AddURL:    "/dashboard/questions/skills/add",
	EditURL:   "/dashboard/questions/skills/edit",
	DeleteURL: "/dashboard/questions/skills/delete",
}

type SkillPageData struct {
	Routes       DashboardRoutes
	SkillRoutes  SkillRoutes
	SkillContext SkillContext
	SkillItems   []SkillListItem
	CancelURL    string
	PageTitle    string
}

type SkillTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultSkillTemplateName = SkillTemplateName{
	AddForm:    "add_form_skill.html",
	EditForm:   "edit_form_skill.html",
	DeleteForm: "delete_form_skill.html",
	Table:      "table_skills.html",
}

var DefaultSkillPathTemplate = "internal/templates/skills/"
