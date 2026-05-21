package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"organization-api/internal/models"
	"organization-api/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
)

type NullableUint struct {
	Set   bool
	Valid bool
	Value uint
}

func (n *NullableUint) UnmarshalJSON(data []byte) error {
	n.Set = true
	if string(data) == "null" {
		n.Valid = false
		return nil
	}
	if err := json.Unmarshal(data, &n.Value); err != nil {
		return err
	}
	n.Valid = true
	return nil
}

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

type DepartmentResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	ParentID  *uint     `json:"parent_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type EmployeeResponse struct {
	ID           uint      `json:"id"`
	DepartmentID uint      `json:"department_id"`
	FullName     string    `json:"full_name"`
	Position     string    `json:"position"`
	HiredAt      *string   `json:"hired_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type DepartmentTreeResponse struct {
	Department DepartmentResponse       `json:"department"`
	Employees  []EmployeeResponse       `json:"employees,omitempty"`
	Children   []DepartmentTreeResponse `json:"children,omitempty"`
}

type CreateDepartmentInput struct {
	Name     string
	ParentID *uint
}

type UpdateDepartmentInput struct {
	Name     *string
	ParentID NullableUint
}

type CreateEmployeeInput struct {
	FullName string
	Position string
	HiredAt  *string
}

type DeleteDepartmentInput struct {
	Mode                   string
	ReassignToDepartmentID *uint
}

func normalize(s string) string { return strings.TrimSpace(s) }

func (s *Service) CreateDepartment(ctx context.Context, in CreateDepartmentInput) (*DepartmentResponse, error) {
	name := normalize(in.Name)
	if err := validateName(name); err != nil {
		return nil, fmt.Errorf("%w: name %s", ErrValidation, err.Error())
	}

	if in.ParentID != nil {
		if _, err := s.repo.FindDepartmentByID(ctx, *in.ParentID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("%w: parent department not found", ErrNotFound)
			}
			return nil, err
		}
	}
	count, err := s.repo.CountSiblingDepartmentByName(ctx, in.ParentID, name, nil)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("%w: department name must be unique within parent", ErrConflict)
	}

	d := &models.Department{Name: name, ParentID: in.ParentID}
	if err := s.repo.CreateDepartment(ctx, d); err != nil {
		return nil, err
	}
	return toDepartmentResponse(d), nil
}

func (s *Service) CreateEmployee(ctx context.Context, departmentID uint, in CreateEmployeeInput) (*EmployeeResponse, error) {
	fullName := normalize(in.FullName)
	position := normalize(in.Position)
	if err := validateName(fullName); err != nil {
		return nil, fmt.Errorf("%w: full_name %s", ErrValidation, err.Error())
	}
	if err := validateName(position); err != nil {
		return nil, fmt.Errorf("%w: position %s", ErrValidation, err.Error())
	}

	if _, err := s.repo.FindDepartmentByID(ctx, departmentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: department not found", ErrNotFound)
		}
		return nil, err
	}

	var hiredAt *time.Time
	if in.HiredAt != nil && strings.TrimSpace(*in.HiredAt) != "" {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*in.HiredAt))
		if err != nil {
			return nil, fmt.Errorf("%w: invalid hired_at date", ErrValidation)
		}
		hiredAt = &t
	}

	e := &models.Employee{DepartmentID: departmentID, FullName: fullName, Position: position, HiredAt: hiredAt}
	if err := s.repo.CreateEmployee(ctx, e); err != nil {
		return nil, err
	}
	return toEmployeeResponse(e), nil
}

func (s *Service) GetDepartment(ctx context.Context, id uint, depth int, includeEmployees bool) (*DepartmentTreeResponse, error) {
	if depth < 0 || depth > 5 {
		return nil, fmt.Errorf("%w: depth must be between 0 and 5", ErrValidation)
	}
	d, err := s.repo.FindDepartmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: department not found", ErrNotFound)
		}
		return nil, err
	}
	return s.buildTree(ctx, d, depth, includeEmployees)
}

func (s *Service) buildTree(ctx context.Context, d *models.Department, depth int, includeEmployees bool) (*DepartmentTreeResponse, error) {
	resp := &DepartmentTreeResponse{Department: *toDepartmentResponse(d)}
	if includeEmployees {
		employees, err := s.repo.FindEmployeesByDepartmentID(ctx, d.ID)
		if err != nil {
			return nil, err
		}
		resp.Employees = make([]EmployeeResponse, 0, len(employees))
		for _, e := range employees {
			resp.Employees = append(resp.Employees, *toEmployeeResponse(&e))
		}
		sort.Slice(resp.Employees, func(i, j int) bool {
			if resp.Employees[i].CreatedAt.Equal(resp.Employees[j].CreatedAt) {
				return resp.Employees[i].FullName < resp.Employees[j].FullName
			}
			return resp.Employees[i].CreatedAt.Before(resp.Employees[j].CreatedAt)
		})
	}
	if depth == 0 {
		return resp, nil
	}
	children, err := s.repo.FindDepartmentsByParentID(ctx, &d.ID)
	if err != nil {
		return nil, err
	}
	resp.Children = make([]DepartmentTreeResponse, 0, len(children))
	for i := range children {
		child := children[i]
		childResp, err := s.buildTree(ctx, &child, depth-1, includeEmployees)
		if err != nil {
			return nil, err
		}
		resp.Children = append(resp.Children, *childResp)
	}
	return resp, nil
}

func (s *Service) UpdateDepartment(ctx context.Context, id uint, in UpdateDepartmentInput) (*DepartmentResponse, error) {
	d, err := s.repo.FindDepartmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: department not found", ErrNotFound)
		}
		return nil, err
	}

	if in.Name != nil {
		name := normalize(*in.Name)
		if err := validateName(name); err != nil {
			return nil, fmt.Errorf("%w: name %s", ErrValidation, err.Error())
		}
		count, err := s.repo.CountSiblingDepartmentByName(ctx, d.ParentID, name, &d.ID)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, fmt.Errorf("%w: department name must be unique within parent", ErrConflict)
		}
		d.Name = name
	}

	if in.ParentID.Set {
		if !in.ParentID.Valid {
			d.ParentID = nil
		} else {
			if in.ParentID.Value == d.ID {
				return nil, fmt.Errorf("%w: department cannot be parent of itself", ErrConflict)
			}
			if _, err := s.repo.FindDepartmentByID(ctx, in.ParentID.Value); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, fmt.Errorf("%w: parent department not found", ErrNotFound)
				}
				return nil, err
			}
			isCycle, err := s.wouldCreateCycle(ctx, d.ID, in.ParentID.Value)
			if err != nil {
				return nil, err
			}
			if isCycle {
				return nil, fmt.Errorf("%w: moving department into its subtree is not allowed", ErrConflict)
			}
			d.ParentID = &in.ParentID.Value
		}
		count, err := s.repo.CountSiblingDepartmentByName(ctx, d.ParentID, d.Name, &d.ID)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, fmt.Errorf("%w: department name must be unique within parent", ErrConflict)
		}
	}

	if err := s.repo.UpdateDepartment(ctx, d); err != nil {
		return nil, err
	}
	return toDepartmentResponse(d), nil
}

func (s *Service) wouldCreateCycle(ctx context.Context, deptID, newParentID uint) (bool, error) {
	current := newParentID
	for {
		if current == deptID {
			return true, nil
		}
		p, err := s.repo.FindDepartmentByID(ctx, current)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return false, err
		}
		if p.ParentID == nil {
			return false, nil
		}
		current = *p.ParentID
	}
}

func (s *Service) DeleteDepartment(ctx context.Context, id uint, in DeleteDepartmentInput) error {
	if in.Mode == "" {
		in.Mode = "cascade"
	}
	if in.Mode != "cascade" && in.Mode != "reassign" {
		return fmt.Errorf("%w: mode must be cascade or reassign", ErrValidation)
	}

	_, err := s.repo.FindDepartmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: department not found", ErrNotFound)
		}
		return err
	}

	if in.Mode == "cascade" {
		return s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			r := s.repo.WithTx(tx)
			return r.DeleteDepartment(ctx, id)
		})
	}

	if in.ReassignToDepartmentID == nil {
		return fmt.Errorf("%w: reassign_to_department_id is required", ErrValidation)
	}
	if *in.ReassignToDepartmentID == id {
		return fmt.Errorf("%w: cannot reassign to the same department", ErrConflict)
	}
	if _, err := s.repo.FindDepartmentByID(ctx, *in.ReassignToDepartmentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: target department not found", ErrNotFound)
		}
		return err
	}

	return s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Employee{}).Where("department_id = ?", id).Update("department_id", *in.ReassignToDepartmentID).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Department{}).Where("parent_id = ?", id).Update("parent_id", nil).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Department{}, id).Error
	})
}

func validateName(v string) error {
	if len(v) < 1 || len(v) > 200 {
		return fmt.Errorf("must be 1..200 characters")
	}
	return nil
}

func toDepartmentResponse(d *models.Department) *DepartmentResponse {
	return &DepartmentResponse{ID: d.ID, Name: d.Name, ParentID: d.ParentID, CreatedAt: d.CreatedAt}
}

func toEmployeeResponse(e *models.Employee) *EmployeeResponse {
	var hiredAt *string
	if e.HiredAt != nil {
		s := e.HiredAt.Format("2006-01-02")
		hiredAt = &s
	}
	return &EmployeeResponse{ID: e.ID, DepartmentID: e.DepartmentID, FullName: e.FullName, Position: e.Position, HiredAt: hiredAt, CreatedAt: e.CreatedAt}
}

