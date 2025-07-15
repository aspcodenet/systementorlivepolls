package data

import (
	"gorm.io/gorm"
)

type AdminUser struct {
	Email      string `gorm:"size:100,uniqueIndex"`
	FromInvite string `gorm:"size:30"`
	AccessKey1 string `gorm:"size:30"`
	SecretKey1 string `gorm:"size:30"`
	AccessKey2 string `gorm:"size:30"`
	SecretKey2 string `gorm:"size:30"`
	Active     bool
	Polls      []Poll `gorm:"foreignKey:AdminUserID"`
	gorm.Model
}
