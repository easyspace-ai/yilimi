package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// writeSSE 写出与 Python _sse_pack 一致的 SSE 帧：event + JSON data
func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b); err != nil {
		return err
	}
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}

// writeSSEDone 结束流：与主站 chat_completions 一致
func writeSSEDone(w http.ResponseWriter, flusher http.Flusher) {
	_, _ = fmt.Fprintf(w, "event: done\ndata: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}
