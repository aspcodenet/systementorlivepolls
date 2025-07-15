package data

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type DbConfig struct {
	Username string
	Password string
	Database string
	Server   string
}

var DB *gorm.DB

func InitDb(dbConfig *DbConfig) {
	var err error

	var url string
	url = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbConfig.Username, dbConfig.Password, dbConfig.Server, dbConfig.Database)

	DB, err = gorm.Open(mysql.Open(url), &gorm.Config{})
	if err != nil {
		panic(err.Error())
	}
	DB.AutoMigrate(&AdminUser{}, &Poll{}, &Question{}, &Vote{}, &Option{})

	seedData(DB)
}

func seedData(DB *gorm.DB) {

}
