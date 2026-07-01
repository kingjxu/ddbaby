package poker

import (
	"context"
	"errors"
	"github.com/kingjxu/ddbaby/dal/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

// 用户激活码表
type UserActiveCode struct {
	Id         int64     `gorm:"column:id;type:bigint(20);primary_key;AUTO_INCREMENT;comment:自增主键ID" json:"id"`
	UserId     string    `gorm:"column:user_id;type:varchar(64);comment:用户ID，可为空" json:"user_id"`
	ActiveCode string    `gorm:"column:active_code;type:varchar(128);comment:激活码;NOT NULL" json:"active_code"`
	CodeType   int       `gorm:"column:code_type;type:tinyint(4);comment:激活码类型：1=按次过期，2=按时间过期;NOT NULL" json:"code_type"`
	InvokeCnt  int64     `gorm:"column:invoke_cnt;type:bigint(20);default:0;comment:激活码调用次数;NOT NULL" json:"invoke_cnt"`
	TotalCnt   int64     `gorm:"column:total_cnt;type:bigint(20);default:0;comment:激活码总次数;NOT NULL" json:"total_cnt"`
	CreateTime time.Time `gorm:"column:create_time;type:datetime;default:CURRENT_TIMESTAMP;comment:创建时间;NOT NULL" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;type:datetime;default:CURRENT_TIMESTAMP;comment:更新时间;NOT NULL" json:"update_time"`
	ExpireAt   int64     `gorm:"column:expire_at;type:bigint(20);comment:过期时间戳" json:"expire_at"`
}

func (m *UserActiveCode) TableName() string {
	return "user_active_code"
}

func GetUserActiveCodeByUserId(ctx context.Context, userId string) (*UserActiveCode, error) {
	var info UserActiveCode
	err := mysql.GetDB(ctx).Debug().Where("user_id = ?", userId).First(&info).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &info, nil
}
func GetUserActiveCodeByCode(ctx context.Context, code string) (*UserActiveCode, error) {
	var info UserActiveCode
	err := mysql.GetDB(ctx).Debug().Where("active_code = ?", code).First(&info).Error
	if err != nil {
		return nil, err
	}
	return &info, nil
}
func UpsertUserActiveCode(ctx context.Context, info *UserActiveCode) error {
	col := []string{"invoke_cnt"}
	if info.CodeType != 0 {
		col = append(col, "expire_at")
	}
	if info.UserId != "" {
		col = append(col, "user_id")
	}
	if info.ExpireAt != 0 {
		col = append(col, "expire_at")
	}
	if info.InvokeCnt != 0 {
		col = append(col, "invoke_cnt")
	}
	if info.TotalCnt != 0 {
		col = append(col, "total_cnt")
	}
	return mysql.GetDB(ctx).Debug().Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns(col),
	}).Debug().Omit("id").Create(&info).Error
}
