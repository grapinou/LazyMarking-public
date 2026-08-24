package tools

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func ServeUserImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}
	filename := r.PathValue("filename")
	if safePathComponent(filename) != nil {
		http.Error(w, "Invalid image name", http.StatusBadRequest)
		return
	}
	owned, err := queries.UserOwnsImage(r.Context(), db.UserOwnsImageParams{
		RequestedImageName: filename,
		UserID:             userID,
	})
	if err != nil {
		http.Error(w, "Unable to verify image", http.StatusInternalServerError)
		return
	}
	if !owned {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(filepath.Join(config.ImageSavePath, filename))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, filename, info.ModTime(), file)
}
