package internal

import (
	"function/pkg"
	"net/http"
)

func (h *Handler) Scanner(w http.ResponseWriter, r *http.Request) {

	

	pkg.HandleResponse(w, "ok", http.StatusOK)
}
