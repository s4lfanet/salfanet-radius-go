package handlers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type UploadHandler struct{ db *gorm.DB }

func NewUploadHandler(db *gorm.DB) *UploadHandler {
	return &UploadHandler{db: db}
}

const uploadDir = "/var/www/salfanet-radius/uploads"

// POST /api/upload/logo — upload company logo
func (h *UploadHandler) UploadLogo(c fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file required"})
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".svg": true}
	if !allowed[ext] {
		return c.Status(400).JSON(fiber.Map{"error": "unsupported file type"})
	}

	filename := fmt.Sprintf("logo-%d%s", time.Now().UnixMilli(), ext)
	dest := filepath.Join(uploadDir, "logos", filename)

	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to open upload"})
	}
	defer src.Close()

	if err2 := os.MkdirAll(filepath.Dir(dest), 0755); err2 != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create directory"})
	}
	dst, err := os.Create(dest)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create file"})
	}
	defer dst.Close()

	if _, err3 := io.Copy(dst, src); err3 != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to write file"})
	}

	url := "/api/uploads/logos/" + filename
	return c.JSON(fiber.Map{"success": true, "url": url, "filename": filename})
}

// POST /api/upload/payment-proof — upload payment proof image
func (h *UploadHandler) UploadPaymentProof(c fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file required"})
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".pdf": true}
	if !allowed[ext] {
		return c.Status(400).JSON(fiber.Map{"error": "unsupported file type"})
	}

	filename := fmt.Sprintf("payment-proof-%d%s", time.Now().UnixMilli(), ext)
	dest := filepath.Join(uploadDir, "payment-proofs", filename)

	src, err2 := file.Open()
	if err2 != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to open upload"})
	}
	defer src.Close()

	if err3 := os.MkdirAll(filepath.Dir(dest), 0755); err3 != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create directory"})
	}
	dst, err4 := os.Create(dest)
	if err4 != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create file"})
	}
	defer dst.Close()

	if _, err5 := io.Copy(dst, src); err5 != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to write file"})
	}

	url := "/api/uploads/payment-proofs/" + filename
	return c.JSON(fiber.Map{"success": true, "url": url, "filename": filename})
}

// POST /api/upload/pppoe-customer — upload customer ID card photo
func (h *UploadHandler) UploadCustomerPhoto(c fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file required"})
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true}
	if !allowed[ext] {
		return c.Status(400).JSON(fiber.Map{"error": "unsupported file type"})
	}

	filename := fmt.Sprintf("customer-%d%s", time.Now().UnixMilli(), ext)
	dest := filepath.Join(uploadDir, "customers", filename)

	src, err2 := file.Open()
	if err2 != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to open upload"})
	}
	defer src.Close()

	if err3 := os.MkdirAll(filepath.Dir(dest), 0755); err3 != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create directory"})
	}
	dst, err4 := os.Create(dest)
	if err4 != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create file"})
	}
	defer dst.Close()

	if _, err5 := io.Copy(dst, src); err5 != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to write file"})
	}

	url := "/api/uploads/customers/" + filename
	return c.JSON(fiber.Map{"success": true, "url": url, "filename": filename})
}

// GET /api/uploads/logos/:filename — serve uploaded logo file
func (h *UploadHandler) ServeLogoFile(c fiber.Ctx) error {
	filename := c.Params("filename")
	// Security: ensure no path traversal
	if strings.Contains(filename, "..") || strings.ContainsAny(filename, "/\\") {
		return c.Status(400).JSON(fiber.Map{"error": "invalid filename"})
	}
	dest := filepath.Join(uploadDir, "logos", filename)
	return c.SendFile(dest)
}

// GET /api/pwa/icon — PWA app icon (supports ?size=192 / ?size=512)
func PwaIcon(c fiber.Ctx) error {
	size := c.Query("size", "192")

	// 1. Try to serve the latest uploaded company logo from uploads/logos/
	logosDir := uploadDir + "/logos"
	if entries, err := os.ReadDir(logosDir); err == nil {
		// Find the most recently uploaded logo file
		var latest string
		for _, e := range entries {
			if !e.IsDir() && (strings.HasSuffix(e.Name(), ".png") || strings.HasSuffix(e.Name(), ".jpg") || strings.HasSuffix(e.Name(), ".webp")) {
				latest = filepath.Join(logosDir, e.Name())
			}
		}
		if latest != "" {
			c.Set("Cache-Control", "public, max-age=3600")
			return c.SendFile(latest)
		}
	}

	// 2. Serve pre-built icons from public/pwa/
	publicBase := "/var/www/salfanet-radius/public/pwa"
	iconFile := publicBase + "/icon-" + size + ".png"
	if _, err := os.Stat(iconFile); err == nil {
		c.Set("Cache-Control", "public, max-age=3600")
		return c.SendFile(iconFile)
	}
	// Fallback to 192
	icon192 := publicBase + "/icon-192.png"
	if _, err := os.Stat(icon192); err == nil {
		c.Set("Cache-Control", "public, max-age=3600")
		return c.SendFile(icon192)
	}

	// 3. Last resort: minimal 1x1 transparent PNG (avoids 404)
	png1x1 := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x62, 0x00, 0x00, 0x00, 0x02,
		0x00, 0x01, 0xE2, 0x21, 0xBC, 0x33, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42,
		0x60, 0x82,
	}
	c.Set("Content-Type", "image/png")
	return c.Send(png1x1)
}
