package models

import "gorm.io/gorm"

// Setting 存储站点级别的键值对设置
type Setting struct {
	gorm.Model
	Key   string `gorm:"type:varchar(255);uniqueIndex"`
	Value string `gorm:"type:text"`
}
