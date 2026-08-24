package tools

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
)

func ServePdf(username, operation string, qcmType config.QCMType, w http.ResponseWriter, r *http.Request) {
	typstName := username + string(qcmType)
	pdfName := strings.TrimSuffix(typstName, filepath.Ext(typstName)) + ".pdf"
	ServePdfNamed(username, operation, pdfName, w, r)
}
