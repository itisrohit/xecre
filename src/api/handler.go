package api

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/itisrohit/xecre/src/engine"
	"github.com/itisrohit/xecre/src/models"
)

//go:embed ui.html
var ui []byte

type Handler struct {
	Engine *engine.DockerEngine
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(ui)
}

func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fail := func(status int, err string) {
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"error": err})
	}
	if r.Method != http.MethodPost {
		fail(http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.ExecutionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128*1024)).Decode(&req); err != nil {
		fail(http.StatusBadRequest, err.Error())
		return
	}
	if (req.Language != "python" && req.Language != "javascript") || strings.TrimSpace(req.Code) == "" {
		fail(http.StatusBadRequest, "Choose Python or JavaScript and provide code")
		return
	}

	result, err := h.Engine.Execute(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, engine.ErrBusy) {
			status = http.StatusTooManyRequests
		}
		if errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusGatewayTimeout
		}
		fail(status, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
