package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/queue"
)

// ScalingHandler handles Phase 5 scaling endpoints.
// cache is either HybridCache (Redis+Memory) or MemoryCache.
type ScalingHandler struct {
	db    *gorm.DB
	cache CacheInterface
	queue *queue.JobQueue
}

// CacheInterface defines the common interface for MemoryCache and HybridCache.
type CacheInterface interface {
	Set(key string, value interface{}, ttl time.Duration)
	Get(key string, dest interface{}) bool
	Delete(key string)
	DeletePattern(prefix string)
	Flush()
	Stats() map[string]interface{}
}

func NewScalingHandler(db *gorm.DB, c CacheInterface, q *queue.JobQueue) *ScalingHandler {
	return &ScalingHandler{db: db, cache: c, queue: q}
}

// ─── Cache Management ────────────────────────────────────────────────────────

// GET /api/scaling/cache/stats — cache statistics
func (h *ScalingHandler) CacheStats(c fiber.Ctx) error {
	if h.cache == nil {
		return c.JSON(fiber.Map{"enabled": false, "message": "Cache not initialized"})
	}
	return c.JSON(fiber.Map{
		"enabled": true,
		"stats":   h.cache.Stats(),
	})
}

// POST /api/scaling/cache/flush — flush all cache entries
func (h *ScalingHandler) CacheFlush(c fiber.Ctx) error {
	if h.cache != nil {
		h.cache.Flush()
	}
	return c.JSON(fiber.Map{"success": true, "message": "Cache flushed"})
}

// POST /api/scaling/cache/invalidate/:prefix — invalidate keys by prefix
func (h *ScalingHandler) CacheInvalidate(c fiber.Ctx) error {
	prefix := c.Params("prefix")
	if h.cache != nil {
		h.cache.DeletePattern(prefix)
	}
	return c.JSON(fiber.Map{"success": true, "message": "Cache invalidated for prefix: " + prefix})
}

// ─── Job Queue Management ────────────────────────────────────────────────────

// GET /api/scaling/queue/stats — queue statistics
func (h *ScalingHandler) QueueStats(c fiber.Ctx) error {
	if h.queue == nil {
		return c.JSON(fiber.Map{"enabled": false, "message": "Queue not initialized"})
	}
	return c.JSON(fiber.Map{
		"enabled": true,
		"stats":   h.queue.Stats(),
	})
}

// ─── Captive Portal ──────────────────────────────────────────────────────────

// GET /api/captive/identify?ip= — identify user by IP address
func (h *ScalingHandler) CaptiveIdentify(c fiber.Ctx) error {
	ip := c.Query("ip")
	if ip == "" {
		return c.Status(400).JSON(fiber.Map{"error": "IP parameter required"})
	}

	// Look up active session by IP in radacct
	var session struct {
		Username      string `json:"username"`
		NasIPAddress  string `json:"nasIPAddress"`
		AcctStartTime string `json:"acctStartTime"`
	}
	result := h.db.Table("radacct").
		Select("username, nasIPAddress, acctStartTime").
		Where("framedIPAddress = ? AND acctstoptime IS NULL", ip).
		First(&session)

	if result.Error != nil {
		return c.JSON(fiber.Map{
			"identified": false,
			"message":    "No active session found for this IP",
		})
	}

	// Get user details
	var user struct {
		ID         string  `json:"id"`
		Name       *string `json:"name"`
		CustomerID *string `json:"customerId"`
		Phone      *string `json:"phone"`
		Status     string  `json:"status"`
	}
	h.db.Table("pppoe_users").
		Select("id, name, customerId, phone, status").
		Where("username = ?", session.Username).
		First(&user)

	// Get unpaid invoices
	var invoices []struct {
		ID        string  `json:"id"`
		InvoiceNo string  `json:"invoiceNo"`
		Amount    float64 `json:"amount"`
		DueDate   string  `json:"dueDate"`
		Status    string  `json:"status"`
	}
	h.db.Table("invoices").
		Select("id, invoiceNo, amount, dueDate, status").
		Where("userId = ? AND status IN ('PENDING','OVERDUE')", user.ID).
		Find(&invoices)

	return c.JSON(fiber.Map{
		"identified": true,
		"username":   session.Username,
		"user":       user,
		"invoices":   invoices,
	})
}

// ─── Rate Limiting Status ────────────────────────────────────────────────────

// GET /api/scaling/rate-limit/status — rate limiting configuration
func (h *ScalingHandler) RateLimitStatus(c fiber.Ctx) error {
	redisMode := "disabled"
	if h.cache != nil {
		if stats, ok := h.cache.Stats()["mode"].(string); ok {
			redisMode = stats
		}
	}
	return c.JSON(fiber.Map{
		"enabled": true,
		"backend": redisMode,
		"global":  fiber.Map{"max": 100, "window": "1m"},
		"auth":    fiber.Map{"max": 5, "window": "1m"},
		"portal":  fiber.Map{"max": 10, "window": "1m"},
		"message": "Rate limiting is configured via Fiber middleware",
	})
}
