package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"hnsw-coursework/internal/indexmanager"
)

// sseHeartbeat keeps idle connections alive and lets clients detect a dead
// server. The content is an SSE comment, ignored by clients.
const sseHeartbeat = 15 * time.Second

func (s *Server) handleIndexEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, codeInternalError, "streaming is not supported")
		return
	}

	// Subscribe before sending headers so no event can be missed between the
	// state snapshot and the first read from the channel.
	sub, st := s.mgr.Subscribe()
	defer s.mgr.Unsubscribe(sub)

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	send := func(eventType string, payload any) bool {
		data, err := json.Marshal(payload)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	sendSnapshot := func(st indexmanager.State) bool {
		switch st.Status {
		case indexmanager.StatusReady:
			return send(indexmanager.EventDone, indexmanager.DoneData{IndexID: st.IndexID})
		case indexmanager.StatusError:
			return send(indexmanager.EventError, indexmanager.ErrorData{Message: st.Error})
		case indexmanager.StatusBuilding:
			return send(indexmanager.EventProgress, indexmanager.ProgressData{
				Progress:  st.Progress,
				Processed: st.Processed,
				Total:     st.Total,
			})
		default:
			if _, err := fmt.Fprint(w, ": idle\n\n"); err != nil {
				return false
			}
			flusher.Flush()
			return true
		}
	}

	if !sendSnapshot(st) {
		return
	}
	if st.Status == indexmanager.StatusReady || st.Status == indexmanager.StatusError {
		return
	}

	ticker := time.NewTicker(sseHeartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-sub.Done:
			return
		case ev, ok := <-sub.C:
			if !ok {
				return
			}
			if !send(ev.Type, ev.Data) {
				return
			}
			if ev.Type != indexmanager.EventProgress {
				return
			}
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
