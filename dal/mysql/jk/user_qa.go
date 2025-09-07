package jk

import (
	"time"
)

// 健康订单表
type UserQa struct {
	ID         uint64    `gorm:"column:id;primary_key;AUTO_INCREMENT"`                  // 自增ID
	JkType     string    `gorm:"column:jk_type;NOT NULL"`                               // 健康问题类型
	OrderID    uint64    `gorm:"column:order_id;NOT NULL"`                              // 订单id
	UserID     string    `gorm:"column:user_id;NOT NULL"`                               // 用户id
	QaList     string    `gorm:"column:qa_list;NOT NULL"`                               // 用户的问题
	CreateTime time.Time `gorm:"column:create_time;default:CURRENT_TIMESTAMP;NOT NULL"` // 创建时间
}

func (m *UserQa) TableName() string {
	return "user_qa"
}
