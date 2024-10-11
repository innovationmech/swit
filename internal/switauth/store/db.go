package store

import (
	"fmt"
	"github.com/innovationmech/swit/internal/switauth/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"sync"
)

var (
	dbConn *gorm.DB
	once   sync.Once
)

func GetDB() *gorm.DB {
	once.Do(func() {
		cfg := config.GetConfig()
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
			cfg.Database.Username, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			panic("fail to connect database")
		}
		dbConn = db
	})
	return dbConn
}
