package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"organization-api/internal/db"
	"organization-api/internal/models"
	"organization-api/internal/repository"
	"organization-api/internal/service"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(gdb); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := repository.New(gdb)
	svc := service.New(repo)
	return New(svc)
}

func TestCreateAndGetDepartmentTree(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/departments/", bytes.NewReader([]byte(`{"name":"HQ"}`)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var dep models.Department
	if err := json.Unmarshal(rec.Body.Bytes(), &dep); err != nil {
		t.Fatalf("unmarshal dept: %v", err)
	}

	employeeBody := []byte(`{"full_name":"Alice Doe","position":"Engineer","hired_at":"2024-01-10"}`)
	req = httptest.NewRequest(http.MethodPost, "/departments/"+strconv.Itoa(int(dep.ID))+"/employees/", bytes.NewReader(employeeBody))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 employee, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/departments/"+strconv.Itoa(int(dep.ID))+"?depth=1&include_employees=true", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"employees"`)) {
		t.Fatalf("expected employees in response: %s", rec.Body.String())
	}
}
