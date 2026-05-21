package models

import "time"

type Department struct {
	ID        uint         `gorm:"primaryKey" json:"id"`
	Name      string       `gorm:"type:varchar(200);not null;index:idx_parent_name,unique" json:"name"`
	ParentID  *uint        `gorm:"index:idx_parent_name,unique" json:"parent_id,omitempty"`
	Parent    *Department  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Children  []Department `gorm:"foreignKey:ParentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Employees []Employee   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	CreatedAt time.Time    `json:"created_at"`
}

type Employee struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	DepartmentID uint       `gorm:"not null;index" json:"department_id"`
	Department   Department `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	FullName     string     `gorm:"type:varchar(200);not null" json:"full_name"`
	Position     string     `gorm:"type:varchar(200);not null" json:"position"`
	HiredAt      *time.Time `gorm:"type:date" json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
}
