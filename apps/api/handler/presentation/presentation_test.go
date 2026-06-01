package presentation_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/apps/api/handler/presentation"
	"context-os/apps/api/response"
)

// TestStatusMethodNotAllowed verifies that a non-GET request to Status returns 405.
func TestStatusMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/presentation/status", nil)

	presentation.Status(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Status() status = %d, want 405", recorder.Code)
	}
}

// TestStatusIncludesSupportedRoles verifies Status reports role and connector capabilities.
func TestStatusIncludesSupportedRoles(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/presentation/status", nil)

	presentation.Status(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Status() status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{"supported_roles", "presentation_layer", "supported_connectors", "filesystem"} {
		if !strings.Contains(body, want) {
			t.Fatalf("Status() body = %s, want substring %q", body, want)
		}
	}
}

// TestFindingsMethodNotAllowed verifies that a non-POST request to Findings returns 405.
func TestFindingsMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/presentation/findings", nil)

	presentation.Findings(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Findings() status = %d, want 405", recorder.Code)
	}
}

// TestFindingsRejectsMalformedJSON verifies malformed JSON returns 400.
func TestFindingsRejectsMalformedJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/presentation/findings", strings.NewReader("{bad json}"))
	req.Header.Set("Content-Type", "application/json")

	presentation.Findings(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Findings() status = %d, want 400", recorder.Code)
	}
}

// TestFindingsRejectsUnknownConnector verifies unsupported connectors are rejected.
func TestFindingsRejectsUnknownConnector(t *testing.T) {
	body := `{"connector":"unknown","uri":"inline.txt","content":"sample"}`
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/presentation/findings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	presentation.Findings(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Findings() status = %d, want 400", recorder.Code)
	}
}

// TestFindingsBuildsRoleOutput verifies graph-backed findings output includes role views, PMO model, and execution evidence.
func TestFindingsBuildsRoleOutput(t *testing.T) {
	body := `{"connector":"filesystem","uri":"inline.txt","content":"frontend expects refundStatus but backend exposes missingRefundState","role":"pmo"}`
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/presentation/findings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	presentation.Findings(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Findings() status = %d, want 200 (body: %s)", recorder.Code, recorder.Body.String())
	}

	var payload response.PresentationFindings
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload.Role != "pmo" {
		t.Fatalf("Role = %q, want pmo", payload.Role)
	}
	if payload.Views.PMO.Role != "pmo" {
		t.Fatalf("Views.PMO.Role = %q, want pmo", payload.Views.PMO.Role)
	}
	if payload.Execution.Enabled != true || payload.Execution.Assistive != true {
		t.Fatalf("Execution = %+v, want enabled assistive evidence", payload.Execution)
	}
	if len(payload.PMO.Facts) == 0 {
		t.Fatalf("PMO.Facts length = %d, want > 0", len(payload.PMO.Facts))
	}
	if payload.TraceID == "" {
		t.Fatal("TraceID = empty, want stable trace id")
	}
}

// TestFindingsCanDisableExecution verifies include_execution=false skips executor call details.
func TestFindingsCanDisableExecution(t *testing.T) {
	body := `{"connector":"filesystem","uri":"inline.txt","content":"frontend displays refundStatus from backend status API","include_execution":false}`
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/presentation/findings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	presentation.Findings(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Findings() status = %d, want 200", recorder.Code)
	}

	var payload response.PresentationFindings
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload.Execution.Enabled {
		t.Fatalf("Execution.Enabled = %v, want false", payload.Execution.Enabled)
	}
}
