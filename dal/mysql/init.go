package mysql

import (
	"context"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var dsn = "root:xujin123@tcp(127.0.0.1:3306)/health?charset=utf8&parseTime=True&loc=Local"

var DB *gorm.DB

func init() {
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
		Logger:                 logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic(err)
	}
}

func GetDB(ctx context.Context) *gorm.DB {
	return DB.WithContext(ctx).Debug()
}
