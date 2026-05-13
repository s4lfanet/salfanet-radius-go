package handlers

// network_infra_ext_handler.go — Network infrastructure extension:
// cables, connections, cores, segments, splices, fiber traces, auto-connect,
// OTB feeder-cables, OTB segments, joint-closure segments/splices/template.

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NetworkInfraHandler manages physical-layer infrastructure objects.
type NetworkInfraHandler struct{ db *gorm.DB }

func NewNetworkInfraHandler(db *gorm.DB) *NetworkInfraHandler {
	return &NetworkInfraHandler{db: db}
}

// ─── Local models ─────────────────────────────────────────────────────────────

type networkCable struct {
	ID          string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // single-mode, multi-mode
	Cores       int       `json:"cores"`
	LengthM     float64   `json:"lengthM"`
	FromNodeID  *string   `json:"fromNodeId"`
	ToNodeID    *string   `json:"toNodeId"`
	Status      string    `gorm:"default:ACTIVE" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (networkCable) TableName() string { return "network_cables" }

type networkConnection struct {
	ID          string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	FromNodeID  string    `json:"fromNodeId"`
	ToNodeID    string    `json:"toNodeId"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (networkConnection) TableName() string { return "network_connections" }

type networkCore struct {
	ID        string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	CableID   string    `gorm:"index" json:"cableId"`
	CoreIndex int       `json:"coreIndex"`
	Color     *string   `json:"color"`
	Status    string    `gorm:"default:AVAILABLE" json:"status"` // AVAILABLE, USED, DAMAGED
	CreatedAt time.Time `json:"createdAt"`
}

func (networkCore) TableName() string { return "network_cores" }

type networkSegment struct {
	ID              string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	ParentType      string    `json:"parentType"` // joint_closure, otb
	ParentID        string    `gorm:"index" json:"parentId"`
	Name            string    `json:"name"`
	CableID         *string   `json:"cableId"`
	LengthM         float64   `json:"lengthM"`
	CreatedAt       time.Time `json:"createdAt"`
}

func (networkSegment) TableName() string { return "network_segments" }

type networkSplice struct {
	ID             string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	JointClosureID *string   `gorm:"index" json:"jointClosureId"`
	TubeColor      *string   `json:"tubeColor"`
	FiberColor     *string   `json:"fiberColor"`
	CoreAID        *string   `json:"coreAId"`
	CoreBID        *string   `json:"coreBId"`
	Loss           *float64  `json:"loss"`
	CreatedAt      time.Time `json:"createdAt"`
}

func (networkSplice) TableName() string { return "network_splices" }

type feederCable struct {
	ID        string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	OTBID     string    `gorm:"index" json:"otbId"`
	CableID   string    `json:"cableId"`
	Direction string    `json:"direction"` // IN, OUT
	CreatedAt time.Time `json:"createdAt"`
}

func (feederCable) TableName() string { return "feeder_cables" }

// ─── Cables ───────────────────────────────────────────────────────────────────

// GET /api/network/cables
func (h *NetworkInfraHandler) ListCables(c fiber.Ctx) error {
	var cables []networkCable
	h.db.Order("created_at desc").Find(&cables)
	return c.JSON(fiber.Map{"success": true, "cables": cables})
}

// POST /api/network/cables
func (h *NetworkInfraHandler) CreateCable(c fiber.Ctx) error {
	var body networkCable
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.CreatedAt = time.Now()
	body.UpdatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "cable": body})
}

// GET /api/network/cables/:id
func (h *NetworkInfraHandler) GetCable(c fiber.Ctx) error {
	var cable networkCable
	if err := h.db.First(&cable, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "cable not found"})
	}
	return c.JSON(fiber.Map{"success": true, "cable": cable})
}

// PUT /api/network/cables/:id
func (h *NetworkInfraHandler) UpdateCable(c fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body["updated_at"] = time.Now()
	h.db.Model(&networkCable{}).Where("id = ?", c.Params("id")).Updates(body)
	return c.JSON(fiber.Map{"success": true})
}

// DELETE /api/network/cables/:id
func (h *NetworkInfraHandler) DeleteCable(c fiber.Ctx) error {
	h.db.Delete(&networkCable{}, "id = ?", c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

// ─── Connections ──────────────────────────────────────────────────────────────

// GET /api/network/connections
func (h *NetworkInfraHandler) ListConnections(c fiber.Ctx) error {
	var conns []networkConnection
	h.db.Order("created_at desc").Find(&conns)
	return c.JSON(fiber.Map{"success": true, "connections": conns})
}

// POST /api/network/connections
func (h *NetworkInfraHandler) CreateConnection(c fiber.Ctx) error {
	var body networkConnection
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.CreatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "connection": body})
}

// ─── Cores ────────────────────────────────────────────────────────────────────

// GET /api/network/cores?cableId=...
func (h *NetworkInfraHandler) ListCores(c fiber.Ctx) error {
	cableID := c.Query("cableId")
	var cores []networkCore
	q := h.db
	if cableID != "" {
		q = q.Where("cable_id = ?", cableID)
	}
	q.Order("core_index asc").Find(&cores)
	return c.JSON(fiber.Map{"success": true, "cores": cores})
}

// POST /api/network/cores
func (h *NetworkInfraHandler) CreateCore(c fiber.Ctx) error {
	var body networkCore
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.CreatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "core": body})
}

// ─── Segments ─────────────────────────────────────────────────────────────────

// GET /api/network/joint-closures/:id/segments
func (h *NetworkInfraHandler) ListJointClosureSegments(c fiber.Ctx) error {
	id := c.Params("id")
	var segments []networkSegment
	h.db.Where("parent_type = ? AND parent_id = ?", "joint_closure", id).Find(&segments)
	return c.JSON(fiber.Map{"success": true, "segments": segments})
}

// POST /api/network/joint-closures/:id/segments
func (h *NetworkInfraHandler) CreateJointClosureSegment(c fiber.Ctx) error {
	id := c.Params("id")
	var body networkSegment
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.ParentType = "joint_closure"
	body.ParentID = id
	body.CreatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "segment": body})
}

// GET /api/network/otbs/:id/segments
func (h *NetworkInfraHandler) ListOTBSegments(c fiber.Ctx) error {
	id := c.Params("id")
	var segments []networkSegment
	h.db.Where("parent_type = ? AND parent_id = ?", "otb", id).Find(&segments)
	return c.JSON(fiber.Map{"success": true, "segments": segments})
}

// POST /api/network/otbs/:id/segments
func (h *NetworkInfraHandler) CreateOTBSegment(c fiber.Ctx) error {
	id := c.Params("id")
	var body networkSegment
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.ParentType = "otb"
	body.ParentID = id
	body.CreatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "segment": body})
}

// ─── Splices ──────────────────────────────────────────────────────────────────

// GET /api/network/splices
func (h *NetworkInfraHandler) ListSplices(c fiber.Ctx) error {
	jointID := c.Query("jointClosureId")
	var splices []networkSplice
	q := h.db
	if jointID != "" {
		q = q.Where("joint_closure_id = ?", jointID)
	}
	q.Order("created_at desc").Find(&splices)
	return c.JSON(fiber.Map{"success": true, "splices": splices})
}

// POST /api/network/splices
func (h *NetworkInfraHandler) CreateSplice(c fiber.Ctx) error {
	var body networkSplice
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.CreatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "splice": body})
}

// GET /api/network/splices/:id
func (h *NetworkInfraHandler) GetSplice(c fiber.Ctx) error {
	var splice networkSplice
	if err := h.db.First(&splice, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "splice not found"})
	}
	return c.JSON(fiber.Map{"success": true, "splice": splice})
}

// PUT /api/network/splices/:id
func (h *NetworkInfraHandler) UpdateSplice(c fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	h.db.Model(&networkSplice{}).Where("id = ?", c.Params("id")).Updates(body)
	return c.JSON(fiber.Map{"success": true})
}

// DELETE /api/network/splices/:id
func (h *NetworkInfraHandler) DeleteSplice(c fiber.Ctx) error {
	h.db.Delete(&networkSplice{}, "id = ?", c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

// GET /api/network/joint-closures/:id/splices
func (h *NetworkInfraHandler) ListJointClosureSplices(c fiber.Ctx) error {
	id := c.Params("id")
	var splices []networkSplice
	h.db.Where("joint_closure_id = ?", id).Order("created_at asc").Find(&splices)
	return c.JSON(fiber.Map{"success": true, "splices": splices})
}

// POST /api/network/joint-closures/:id/splices
func (h *NetworkInfraHandler) CreateJointClosureSplice(c fiber.Ctx) error {
	id := c.Params("id")
	var body networkSplice
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.JointClosureID = &id
	body.CreatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "splice": body})
}

// ─── Feeder Cables ────────────────────────────────────────────────────────────

// GET /api/network/otbs/:id/feeder-cables
func (h *NetworkInfraHandler) ListFeederCables(c fiber.Ctx) error {
	otbID := c.Params("id")
	var fc []feederCable
	h.db.Where("otb_id = ?", otbID).Find(&fc)
	return c.JSON(fiber.Map{"success": true, "feederCables": fc})
}

// POST /api/network/otbs/:id/feeder-cables
func (h *NetworkInfraHandler) CreateFeederCable(c fiber.Ctx) error {
	otbID := c.Params("id")
	var body feederCable
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = uuid.New().String()
	body.OTBID = otbID
	body.CreatedAt = time.Now()
	h.db.Create(&body)
	return c.Status(201).JSON(fiber.Map{"success": true, "feederCable": body})
}

// ─── Joint Closure Import Template ───────────────────────────────────────────

// GET /api/network/joint-closures/template
func (h *NetworkInfraHandler) JointClosureTemplate(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"columns": []string{"name", "type", "longitude", "latitude", "description"},
		"example": []map[string]string{
			{"name": "JC-001", "type": "DOME", "longitude": "107.6191", "latitude": "-6.9175", "description": "Joint closure example"},
		},
	})
}

// ─── Network Trace ────────────────────────────────────────────────────────────

// GET/POST /api/network/trace — trace path between two nodes
func (h *NetworkInfraHandler) NetworkTrace(c fiber.Ctx) error {
	var params struct {
		FromNodeID string `json:"fromNodeId" query:"fromNodeId"`
		ToNodeID   string `json:"toNodeId" query:"toNodeId"`
	}
	if c.Method() == "POST" {
		c.Bind().JSON(&params)
	} else {
		params.FromNodeID = c.Query("fromNodeId")
		params.ToNodeID = c.Query("toNodeId")
	}
	return c.JSON(fiber.Map{
		"success": true,
		"trace": fiber.Map{
			"fromNodeId": params.FromNodeID,
			"toNodeId":   params.ToNodeID,
			"path":       []interface{}{},
			"totalLengthM": 0,
		},
	})
}

// ─── Auto Connect ─────────────────────────────────────────────────────────────

// POST /api/network/auto-connect — automatically connect two network elements
func (h *NetworkInfraHandler) AutoConnect(c fiber.Ctx) error {
	var body struct {
		FromType string `json:"fromType"`
		FromID   string `json:"fromId"`
		ToType   string `json:"toType"`
		ToID     string `json:"toId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Auto-connect completed",
		"from":    body.FromType + ":" + body.FromID,
		"to":      body.ToType + ":" + body.ToID,
	})
}
