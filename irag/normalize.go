package irag

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func shouldPassThroughRaw(path, contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if strings.HasPrefix(ct, "image/") || strings.HasPrefix(ct, "audio/") || strings.HasPrefix(ct, "video/") {
		return true
	}
	if strings.Contains(ct, "application/octet-stream") || strings.Contains(ct, "application/pdf") || strings.Contains(ct, "application/zip") {
		return true
	}
	if strings.HasPrefix(path, "/v1/download/") {
		return true
	}
	if strings.HasPrefix(path, "/v1/ai/image/") {
		return true
	}
	return false
}

func buildEnvelope(status int, data any, meta map[string]any, errMsg string) ([]byte, error) {
	envelope := map[string]any{
		"ok":        status >= 200 && status < 300,
		"code":      status,
		"meta":      meta,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if status >= 200 && status < 300 {
		envelope["data"] = data
	} else {
		envelope["error"] = map[string]any{
			"message": errMsg,
		}
	}
	return json.Marshal(envelope)
}

func normalizeJSONBody(body []byte) any {
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return string(bytes.TrimSpace(body))
	}
	switch value := decoded.(type) {
	case map[string]any:
		if result, ok := value["result"]; ok {
			return result
		}
		if data, ok := value["data"]; ok {
			return data
		}
		if response, ok := value["response"]; ok {
			return response
		}
		if items, ok := value["items"]; ok {
			return items
		}
		return value
	default:
		return decoded
	}
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
