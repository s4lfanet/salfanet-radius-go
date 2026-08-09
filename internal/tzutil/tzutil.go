// Package tzutil provides timezone-aware time utilities.
// It reads the timezone from the company settings in the database (single source of truth),
// falling back to APP_TIMEZONE from the environment config.
package tzutil

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

var (
	mu       sync.RWMutex
	location *time.Location
	tzName   string
)

// Init sets the initial timezone from the config (APP_TIMEZONE env var).
// This is called once at startup before the DB is available.
func Init(cfgLoc *time.Location, cfgName string) {
	mu.Lock()
	defer mu.Unlock()
	location = cfgLoc
	tzName = cfgName
	log.Info().Str("timezone", tzName).Msg("tzutil: initialized from config")
}

// LoadFromDB reads the timezone from the company settings table and updates the global timezone.
// This should be called after DB init and whenever company settings are saved.
func LoadFromDB(db *gorm.DB) {
	var tz string
	if err := db.Raw("SELECT timezone FROM companies LIMIT 1").Scan(&tz).Error; err != nil || tz == "" {
		log.Debug().Msg("tzutil: no timezone in companies table, keeping config default")
		return
	}
	SetTimezone(tz)
}

// SetTimezone updates the global timezone. Called when company settings are saved.
func SetTimezone(tz string) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		log.Error().Err(err).Str("timezone", tz).Msg("tzutil: invalid timezone, keeping current")
		return
	}
	mu.Lock()
	defer mu.Unlock()
	location = loc
	tzName = tz
	log.Info().Str("timezone", tzName).Msg("tzutil: timezone updated")
}

// Now returns the current time in the configured timezone.
// This is the single source of truth for "current time" in the Go backend.
func Now() time.Time {
	mu.RLock()
	defer mu.RUnlock()
	if location == nil {
		return time.Now()
	}
	return time.Now().In(location)
}

// Location returns the configured timezone location.
func Location() *time.Location {
	mu.RLock()
	defer mu.RUnlock()
	if location == nil {
		loc, _ := time.LoadLocation("Asia/Jakarta")
		return loc
	}
	return location
}

// Name returns the configured timezone name (e.g. "Asia/Jakarta").
func Name() string {
	mu.RLock()
	defer mu.RUnlock()
	if tzName == "" {
		return "Asia/Jakarta"
	}
	return tzName
}

// StartOfToday returns midnight of the current day in the configured timezone.
func StartOfToday() time.Time {
	now := Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// EndOfToday returns 23:59:59.999 of the current day in the configured timezone.
func EndOfToday() time.Time {
	now := Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999, now.Location())
}

// Format formats a time in the configured timezone using the given layout.
func Format(t time.Time, layout string) string {
	return t.In(Location()).Format(layout)
}

// FormatNow returns the current time formatted with the given layout in the configured timezone.
func FormatNow(layout string) string {
	return Format(Now(), layout)
}

// InTZ converts a time to the configured timezone.
func InTZ(t time.Time) time.Time {
	return t.In(Location())
}
