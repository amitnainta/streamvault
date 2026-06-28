package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/amitnainta/streamvault/internal/scheduler"
)

type TaskHandler struct {
	sched *scheduler.Scheduler
	log   *zap.Logger
}

func NewTaskHandler(sched *scheduler.Scheduler, log *zap.Logger) *TaskHandler {
	return &TaskHandler{sched: sched, log: log}
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	tasks := h.sched.ListTasks()
	type taskView struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Schedule string `json:"schedule"`
		Status   string `json:"status"`
		LastRun  string `json:"last_run,omitempty"`
		LastErr  string `json:"last_error,omitempty"`
	}
	var out []taskView
	for _, t := range tasks {
		status, lastRun, lastErr := h.sched.GetStatus(t.ID)
		v := taskView{
			ID:       t.ID,
			Name:     t.Name,
			Schedule: t.Schedule,
			Status:   string(status),
			LastErr:  lastErr,
		}
		if !lastRun.IsZero() {
			v.LastRun = lastRun.Format(time.RFC3339)
		}
		out = append(out, v)
	}
	if out == nil {
		out = []taskView{}
	}
	writeJSON(w, 200, out)
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	status, lastRun, lastErr := h.sched.GetStatus(id)
	if status == "" {
		writeError(w, 404, "task not found")
		return
	}
	writeJSON(w, 200, map[string]any{
		"id":         id,
		"status":     string(status),
		"last_run":   lastRun,
		"last_error": lastErr,
	})
}

func (h *TaskHandler) RunNow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !h.sched.RunNow(id) {
		writeError(w, 404, "task not found")
		return
	}
	writeJSON(w, 202, map[string]string{"status": "triggered", "task_id": id})
}
