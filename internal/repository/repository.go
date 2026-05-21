package repository

import (
	"context"
	"errors"

	"organization-api/internal/models"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) DB() *gorm.DB { return r.db }

func (r *Repository) WithTx(tx *gorm.DB) *Repository { return &Repository{db: tx} }

func (r *Repository) CreateDepartment(ctx context.Context, d *models.Department) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *Repository) UpdateDepartment(ctx context.Context, d *models.Department) error {
	return r.db.WithContext(ctx).Save(d).Error
}

func (r *Repository) DeleteDepartment(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Department{}, id).Error
}

func (r *Repository) FindDepartmentByID(ctx context.Context, id uint) (*models.Department, error) {
	var d models.Department
	err := r.db.WithContext(ctx).First(&d, id).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repository) FindDepartmentsByParentID(ctx context.Context, parentID *uint) ([]models.Department, error) {
	var ds []models.Department
	q := r.db.WithContext(ctx).Order("created_at asc, name asc")
	if parentID == nil {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", *parentID)
	}
	if err := q.Find(&ds).Error; err != nil {
		return nil, err
	}
	return ds, nil
}

func (r *Repository) FindEmployeesByDepartmentID(ctx context.Context, departmentID uint) ([]models.Employee, error) {
	var es []models.Employee
	if err := r.db.WithContext(ctx).Where("department_id = ?", departmentID).Find(&es).Error; err != nil {
		return nil, err
	}
	return es, nil
}

func (r *Repository) CreateEmployee(ctx context.Context, e *models.Employee) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *Repository) CountSiblingDepartmentByName(ctx context.Context, parentID *uint, name string, excludeID *uint) (int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Department{}).Where("name = ?", name)
	if parentID == nil {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", *parentID)
	}
	if excludeID != nil {
		q = q.Where("id <> ?", *excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) ListChildrenIDs(ctx context.Context, parentID uint) ([]uint, error) {
	var ids []uint
	if err := r.db.WithContext(ctx).Model(&models.Department{}).Where("parent_id = ?", parentID).Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }
