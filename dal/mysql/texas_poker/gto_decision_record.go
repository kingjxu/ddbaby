package poker

import (
	"context"
	"time"

	"github.com/kingjxu/ddbaby/dal/mysql"
)

// GtoDecisionRecord stores the exact request and normalized response for a
// successful decision returned to the client.
type GtoDecisionRecord struct {
	ID               uint64    `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id"`
	UID              string    `gorm:"column:uid;type:varchar(128);not null;index:idx_gto_decision_uid" json:"uid"`
	DecisionTime     time.Time `gorm:"column:decision_time;type:datetime(3);not null;index:idx_gto_decision_time" json:"decision_time"`
	HeroCards        string    `gorm:"column:hero_cards;type:json;not null" json:"hero_cards"`
	Position         string    `gorm:"column:position;type:varchar(32);not null" json:"position"`
	Stage            string    `gorm:"column:stage;type:varchar(16);not null" json:"stage"`
	DecisionRequest  string    `gorm:"column:decision_request;type:json;not null" json:"decision_request"`
	DecisionResponse string    `gorm:"column:decision_response;type:json;not null" json:"decision_response"`
	FinalResponse    string    `gorm:"column:final_response;type:json;not null" json:"final_response"`
}

func (GtoDecisionRecord) TableName() string {
	return "gto_decision_record"
}

func CreateGtoDecisionRecord(ctx context.Context, record *GtoDecisionRecord) error {
	return mysql.GetDB(ctx).Create(record).Error
}
