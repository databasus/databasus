package authenticator

import (
	"time"

	"github.com/google/uuid"
)

const (
	maxWrongCodesPerHour int64 = 10
	wrongCodeWindow            = time.Hour
)

type SecondFactorFailure struct {
	ID       uuid.UUID `gorm:"column:id"`
	UserID   uuid.UUID `gorm:"column:user_id"`
	FailedAt time.Time `gorm:"column:failed_at"`
}

func (SecondFactorFailure) TableName() string {
	return "users_second_factor_failures"
}
