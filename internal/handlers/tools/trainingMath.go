package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"html/template"
	"os/exec"
	"strings"
	"time"

	"github.com/grapinou/LazyMarking/internal/mathcontent"
)

// RenderTrainingText keeps prose as escaped HTML and compiles only validated math.
func RenderTrainingText(ctx context.Context, value string) (template.HTML, error) {
	segments, err := mathcontent.Parse(value)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, segment := range segments {
		if !segment.Math {
			out.WriteString(strings.ReplaceAll(html.EscapeString(segment.Text), "\n", "<br>"))
			continue
		}
		svg, err := renderTrainingFormula(ctx, segment.Text)
		if err != nil {
			return "", err
		}
		out.WriteString(`<img class="training-math" alt="`)
		out.WriteString(html.EscapeString(segment.Text))
		out.WriteString(`" src="data:image/svg+xml;base64,`)
		out.WriteString(base64.StdEncoding.EncodeToString(svg))
		out.WriteString(`">`)
	}
	return template.HTML(out.String()), nil
}

func renderTrainingFormula(ctx context.Context, formula string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	source := "#set page(width: auto, height: auto, margin: 0pt)\n#set text(size: 15pt)\n$" + formula + "$\n"
	cmd := exec.CommandContext(ctx, "typst", "compile", "--format", "svg", "-", "-")
	cmd.Stdin = strings.NewReader(source)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	svg, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("formule Typst impossible à composer : %s", strings.TrimSpace(stderr.String()))
	}
	if !bytes.HasPrefix(svg, []byte("<svg")) {
		return nil, fmt.Errorf("sortie SVG Typst invalide")
	}
	return svg, nil
}
