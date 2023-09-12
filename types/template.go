package types

import (
	"strings"

	"github.com/hoisie/mustache"
)

type Template struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	UserId  string `json:"user_id" boltholdIndex:"UserId"`
	Content string `json:"content"`
}

func (t *Template) FillContent(vars map[string]interface{}) string {
	content := t.Content
	return mustache.Render(content, vars)
}

func (t *Template) GuessIsHtml() bool {
	return strings.Contains(t.Content, "<html")
}
