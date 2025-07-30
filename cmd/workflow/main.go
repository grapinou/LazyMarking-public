package main

import (
	"github.com/grapinou/LazyMarking/internal/workflow"
)

func main() {
	baseURL := "http://localhost:8080"

	workflow.GetRegisterWF(baseURL)
	workflow.PostRegisterWF("Sighto", "aa.com", "aa", baseURL)
	workflow.LoginWf(baseURL)
}
