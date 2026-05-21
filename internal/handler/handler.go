package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"organization-api/internal/httputil"
	"organization-api/internal/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

type createDepartmentRequest struct {
	Name     string `json:"name"`
	ParentID *uint  `json:"parent_id"`
}

type createEmployeeRequest struct {
	FullName string  `json:"full_name"`
	Position string  `json:"position"`
	HiredAt  *string `json:"hired_at"`
}

type patchDepartmentRequest struct {
	Name     *string              `json:"name"`
	ParentID service.NullableUint `json:"parent_id"`
}

type deleteDepartmentQuery struct {
	Mode                   string
	ReassignToDepartmentID *uint
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	path := strings.Trim(r.URL.Path, "/")
	parts := []string{}
	if path != "" {
		parts = strings.Split(path, "/")
	}

	if len(parts) == 1 && parts[0] == "departments" {
		switch r.Method {
		case http.MethodPost:
			h.createDepartment(w, r)
		default:
			httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if len(parts) == 2 && parts[0] == "departments" {
		id, err := parseUint(parts[1])
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid department id")
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.getDepartment(w, r, id)
		case http.MethodPatch:
			h.patchDepartment(w, r, id)
		case http.MethodDelete:
			h.deleteDepartment(w, r, id)
		default:
			httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if len(parts) == 3 && parts[0] == "departments" && parts[2] == "employees" {
		id, err := parseUint(parts[1])
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid department id")
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.createEmployee(w, r, id)
		return
	}

	httputil.WriteError(w, http.StatusNotFound, "not found")
}

func (h *Handler) createDepartment(w http.ResponseWriter, r *http.Request) {
	var req createDepartmentRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	resp, err := h.svc.CreateDepartment(r.Context(), service.CreateDepartmentInput{Name: req.Name, ParentID: req.ParentID})
	if err != nil {
		h.writeSvcError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) createEmployee(w http.ResponseWriter, r *http.Request, departmentID uint) {
	var req createEmployeeRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	resp, err := h.svc.CreateEmployee(r.Context(), departmentID, service.CreateEmployeeInput{FullName: req.FullName, Position: req.Position, HiredAt: req.HiredAt})
	if err != nil {
		h.writeSvcError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) getDepartment(w http.ResponseWriter, r *http.Request, id uint) {
	depth := 1
	if s := r.URL.Query().Get("depth"); s != "" {
		d, err := strconv.Atoi(s)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid depth")
			return
		}
		depth = d
	}
	includeEmployees := true
	if s := r.URL.Query().Get("include_employees"); s != "" {
		b, err := strconv.ParseBool(s)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid include_employees")
			return
		}
		includeEmployees = b
	}
	resp, err := h.svc.GetDepartment(r.Context(), id, depth, includeEmployees)
	if err != nil {
		h.writeSvcError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) patchDepartment(w http.ResponseWriter, r *http.Request, id uint) {
	var req patchDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	resp, err := h.svc.UpdateDepartment(r.Context(), id, service.UpdateDepartmentInput{Name: req.Name, ParentID: req.ParentID})
	if err != nil {
		h.writeSvcError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) deleteDepartment(w http.ResponseWriter, r *http.Request, id uint) {
	mode := r.URL.Query().Get("mode")
	var reassignID *uint
	if s := r.URL.Query().Get("reassign_to_department_id"); s != "" {
		v, err := parseUint(s)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid reassign_to_department_id")
			return
		}
		reassignID = &v
	}
	err := h.svc.DeleteDepartment(r.Context(), id, service.DeleteDepartmentInput{Mode: mode, ReassignToDepartmentID: reassignID})
	if err != nil {
		h.writeSvcError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeSvcError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		httputil.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrConflict):
		httputil.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrValidation):
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func parseUint(s string) (uint, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	return uint(v), err
}
