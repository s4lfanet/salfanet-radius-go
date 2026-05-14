package handlers

// fcm_handler.go — Firebase Cloud Messaging (FCM) token registration + test
// POST /api/fcm/token — register a device FCM token
// POST /api/fcm/test — send a test FCM notification

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FCMHandler handles FCM device token registration and test notifications.
type FCMHandler struct{ db *gorm.DB }

func NewFCMHandler(db *gorm.DB) *FCMHandler {
	return &FCMHandler{db: db}
}

type fcmToken struct {
	ID        string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	UserID    string    `gorm:"index" json:"userId"`
	Token     string    `gorm:"uniqueIndex;type:text" json:"token"`
	Platform  string    `json:"platform"` // android, ios, web
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (fcmToken) TableName() string { return "fcm_tokens" }

// POST /api/fcm/token — register or update a device token
func (h *FCMHandler) RegisterToken(c fiber.Ctx) error {
	var body struct {
		UserID   string `json:"userId"`
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Token == "" {
		return c.Status(400).JSON(fiber.Map{"error": "token required"})
	}
	now := time.Now()
	var existing fcmToken
	if h.db.Where("token = ?", body.Token).First(&existing).Error == nil {
		h.db.Model(&existing).Updates(map[string]interface{}{
			"userId":    body.UserID,
			"platform":   body.Platform,
			"updatedAt": now,
		})
		return c.JSON(fiber.Map{"success": true, "message": "FCM token updated"})
	}
	record := fcmToken{
		ID:        uuid.New().String(),
		UserID:    body.UserID,
		Token:     body.Token,
		Platform:  body.Platform,
		CreatedAt: now,
		UpdatedAt: now,
	}
	h.db.Create(&record)
	return c.Status(201).JSON(fiber.Map{"success": true, "message": "FCM token registered"})
}

// POST /api/fcm/test — send a test FCM notification (no actual FCM SDK — stub)
func (h *FCMHandler) Test(c fiber.Ctx) error {
	var body struct {
		Token   string `json:"token"`
		Title   string `json:"title"`
		Message string `json:"message"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Token == "" {
		return c.Status(400).JSON(fiber.Map{"error": "token required"})
	}
	// Stub: in production integrate with FCM HTTP API / firebase-admin SDK
	preview := body.Token
	if len(preview) > 20 {
		preview = preview[:20] + "..."
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "FCM test notification queued (stub)",
		"token":   preview,
	})
}
