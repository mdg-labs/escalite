package graph

import (
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func validateExpoPushToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("expoPushToken is required")
	}
	if len(token) > 512 {
		return errors.New("expoPushToken is too long")
	}
	if !strings.HasPrefix(token, "ExponentPushToken[") && !strings.HasPrefix(token, "ExpoPushToken[") {
		return errors.New("expoPushToken must be a valid Expo push token")
	}
	return nil
}

func optionalTextField(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: trimmed, Valid: true}
}

func mobileDeviceFromDB(device db.MobileDevice) *model.MobileDevice {
	var platform *string
	if device.Platform.Valid {
		value := device.Platform.String
		platform = &value
	}

	var deviceLabel *string
	if device.DeviceLabel.Valid {
		value := device.DeviceLabel.String
		deviceLabel = &value
	}

	var revokedAt *time.Time
	if device.RevokedAt.Valid {
		t := device.RevokedAt.Time.UTC()
		revokedAt = &t
	}

	return &model.MobileDevice{
		ID:               device.ID.String(),
		Platform:         platform,
		DeviceLabel:      deviceLabel,
		PushTokenPrefix:  device.PushTokenPrefix,
		RevokedAt:        revokedAt,
		LastRegisteredAt: timeFromDB(device.LastRegisteredAt),
		CreatedAt:        timeFromDB(device.CreatedAt),
		UpdatedAt:        timeFromDB(device.UpdatedAt),
	}
}

func mobileDevicesFromDB(devices []db.MobileDevice) []*model.MobileDevice {
	result := make([]*model.MobileDevice, 0, len(devices))
	for _, device := range devices {
		result = append(result, mobileDeviceFromDB(device))
	}
	return result
}

func pushTokenPrefix(token string) string {
	return auth.TokenPrefix(strings.TrimSpace(token))
}
