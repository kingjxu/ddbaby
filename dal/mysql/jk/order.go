package jk

import (
	"context"
	"github.com/kingjxu/ddbaby/dal/mysql"
	"time"
)

// 健康订单表
type JkOrder struct {
	ID         uint64    `gorm:"column:id;primary_key;AUTO_INCREMENT"`                  // 自增ID
	JkType     string    `gorm:"column:jk_type;NOT NULL"`                               // 健康问题类型
	OrderID    string    `gorm:"column:order_id;NOT NULL"`                              // 订单id
	UserID     string    `gorm:"column:user_id;NOT NULL"`                               // 用户id
	Amount     int       `gorm:"column:amount;NOT NULL"`                                // 金额:单位分
	Status     int       `gorm:"column:status;NOT NULL"`                                // 支付状态：10待支付,20已支付,30已退款
	Seq        int       `gorm:"column:seq;NOT NULL"`                                   // 支付的次序
	WxOpenID   string    `gorm:"column:wx_open_id;NOT NULL"`                            // 微信id
	WxOrderID  string    `gorm:"column:wx_order_id;NOT NULL"`                           // 微信订单id
	CreateTime time.Time `gorm:"column:create_time;default:CURRENT_TIMESTAMP;NOT NULL"` // 创建时间
	UpdateTime time.Time `gorm:"column:update_time;default:CURRENT_TIMESTAMP;NOT NULL"` // 更新时间
	QaItems    string    `gorm:"column:qa_items;NOT NULL"`                              // 用户的问答
	BdVid      string    `gorm:"column:bd_vid;NOT NULL"`                                // 百度投放标识
}

func (m *JkOrder) TableName() string {
	return "jk_order"
}

func CreateOrder(ctx context.Context, order *JkOrder) error {
	return mysql.GetDB(ctx).Create(order).Error
}

func UpdateOrderPaySuccess(ctx context.Context, orderID, openID, wxTransID string) error {
	updates := map[string]interface{}{
		"wx_open_id":  openID,
		"wx_order_id": wxTransID,
		"status":      20,
	}
	return mysql.GetDB(ctx).Model(&JkOrder{}).Where("order_id = ?", orderID).Updates(updates).Error
}

func GetOrderByOrderID(ctx context.Context, orderID string) (*JkOrder, error) {
	var order *JkOrder
	err := mysql.GetDB(ctx).Where("order_id = ?", orderID).First(&order).Error
	return order, err

}
