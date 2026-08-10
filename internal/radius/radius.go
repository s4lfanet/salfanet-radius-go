// Package radius provides direct MySQL-based FreeRADIUS management.
// It does NOT use the RADIUS protocol — it writes directly to the shared DB tables.
package radius

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/s4lfanet/salfanet-radius-go/internal/db/models"
)

// Service handles FreeRADIUS operations via direct DB writes.
type Service struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Service { return &Service{db: db} }

// SetPassword sets or replaces the Cleartext-Password in radcheck.
func (s *Service) SetPassword(username, password string) error {
	return s.db.Exec(`
		INSERT INTO radcheck (username, attribute, op, value)
		VALUES (?, 'Cleartext-Password', ':=', ?)
		ON DUPLICATE KEY UPDATE value = VALUES(value)
	`, username, password).Error
}

// Isolate moves user to isolir group for restricted access (user can still login
// but gets limited access via the isolir group profile).
func (s *Service) Isolate(username string) error {
	// Remove any Auth-Type reject
	if err := s.db.Where("username = ? AND attribute = ?", username, "Auth-Type").
		Delete(&models.Radcheck{}).Error; err != nil {
		return fmt.Errorf("remove auth-type: %w", err)
	}
	// Remove reject reply message
	if err := s.db.Where("username = ? AND attribute = ?", username, "Reply-Message").
		Delete(&models.Radreply{}).Error; err != nil {
		return fmt.Errorf("remove reply-message: %w", err)
	}
	// Move to isolir group
	s.db.Where("username = ?", username).Delete(&models.Radusergroup{})
	if err := s.SetGroup(username, "isolir"); err != nil {
		return fmt.Errorf("set isolir group: %w", err)
	}
	// Remove static IP (user will get IP from pool-isolir)
	s.db.Where("username = ? AND attribute = ?", username, "Framed-IP-Address").
		Delete(&models.Radreply{})
	// Close active sessions
	s.db.Exec(`UPDATE radacct SET acctstoptime = NOW(), acctterminatecause = 'Admin-Reset'
		WHERE username = ? AND acctstoptime IS NULL`, username)
	log.Debug().Str("username", username).Msg("radius: isolate user ok")
	return nil
}

// Activate restores the real password and removes any reject markers.
func (s *Service) Activate(username, password string) error {
	// Remove Auth-Type reject if present
	s.db.Where("username = ? AND attribute = ?", username, "Auth-Type").Delete(&models.Radcheck{})
	// Remove reject reply message
	s.db.Where("username = ? AND attribute = ?", username, "Reply-Message").Delete(&models.Radreply{})
	// Restore password
	return s.SetPassword(username, password)
}

// SetRateLimit sets or updates Mikrotik-Rate-Limit in radreply.
func (s *Service) SetRateLimit(username, rateLimit string) error {
	return s.db.Exec(`
		INSERT INTO radreply (username, attribute, op, value)
		VALUES (?, 'Mikrotik-Rate-Limit', '=', ?)
		ON DUPLICATE KEY UPDATE value = VALUES(value)
	`, username, rateLimit).Error
}

// SetGroup sets or updates radusergroup mapping.
func (s *Service) SetGroup(username, groupname string) error {
	return s.db.Exec(`
		INSERT INTO radusergroup (username, groupname, priority)
		VALUES (?, ?, 1)
		ON DUPLICATE KEY UPDATE groupname = VALUES(groupname)
	`, username, groupname).Error
}

// DeleteUser removes all radius entries for a username.
func (s *Service) DeleteUser(username string) error {
	if err := s.db.Where("username = ?", username).Delete(&models.Radcheck{}).Error; err != nil {
		return err
	}
	if err := s.db.Where("username = ?", username).Delete(&models.Radreply{}).Error; err != nil {
		return err
	}
	if err := s.db.Where("username = ?", username).Delete(&models.Radusergroup{}).Error; err != nil {
		return err
	}
	log.Debug().Str("username", username).Msg("radius: delete user ok")
	return nil
}

// RestoreUser restores an isolated user in RADIUS:
// 1. Remove reject markers (Auth-Type, NAS-IP-Address from radcheck)
// 2. Remove isolation Reply-Message from radreply
// 3. Restore radusergroup to the user's profile group
// 4. Restore static IP if provided
func (s *Service) RestoreUser(username, groupName string, ipAddress *string) error {
	// 1. Remove reject markers
	if err := s.db.Where("username = ? AND attribute IN ?", username, []string{"Auth-Type", "NAS-IP-Address"}).
		Delete(&models.Radcheck{}).Error; err != nil {
		return fmt.Errorf("remove reject markers: %w", err)
	}

	// 2. Remove isolation reply messages
	if err := s.db.Where("username = ? AND attribute = ?", username, "Reply-Message").
		Delete(&models.Radreply{}).Error; err != nil {
		return fmt.Errorf("remove reply-message: %w", err)
	}

	// 3. Restore radusergroup
	s.db.Where("username = ?", username).Delete(&models.Radusergroup{})
	if groupName != "" {
		if err := s.SetGroup(username, groupName); err != nil {
			return fmt.Errorf("restore group: %w", err)
		}
	}

	// 4. Restore static IP
	s.db.Where("username = ? AND attribute = ?", username, "Framed-IP-Address").Delete(&models.Radreply{})
	if ipAddress != nil && *ipAddress != "" {
		if err := s.db.Exec(`
			INSERT INTO radreply (username, attribute, op, value)
			VALUES (?, 'Framed-IP-Address', '=', ?)
		`, username, *ipAddress).Error; err != nil {
			return fmt.Errorf("restore ip: %w", err)
		}
	}

	log.Debug().Str("username", username).Str("group", groupName).Msg("radius: restore user ok")
	return nil
}

// UpsertUser sets up all radius entries for a new/updated PPPoE user.
func (s *Service) UpsertUser(username, password, rateLimit, groupname string) error {
	if err := s.SetPassword(username, password); err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	if rateLimit != "" {
		if err := s.SetRateLimit(username, rateLimit); err != nil {
			return fmt.Errorf("set rate limit: %w", err)
		}
	}
	if groupname != "" {
		if err := s.SetGroup(username, groupname); err != nil {
			return fmt.Errorf("set group: %w", err)
		}
	}
	log.Debug().Str("username", username).Str("group", groupname).Msg("radius: upsert user ok")
	return nil
}

// ActiveSessions returns currently-online sessions from radacct.
func (s *Service) ActiveSessions() ([]models.Radacct, error) {
	var sessions []models.Radacct
	err := s.db.Where("acctstoptime IS NULL").
		Order("acctstarttime DESC").
		Limit(500).
		Find(&sessions).Error
	return sessions, err
}

// SessionByUsername returns the latest active session for a user.
func (s *Service) SessionByUsername(username string) (*models.Radacct, error) {
	var s2 models.Radacct
	err := s.db.Where("username = ? AND acctstoptime IS NULL", username).
		Order("acctstarttime DESC").
		First(&s2).Error
	if err != nil {
		return nil, err
	}
	return &s2, nil
}
