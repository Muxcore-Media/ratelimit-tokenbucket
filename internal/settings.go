package internal

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Muxcore-Media/core/pkg/contracts"
)

func (m *Module) Settings() []contracts.SettingDef {
	return m.settingsDefs()
}

func (m *Module) UpdateSetting(key, value string) error {
	return m.updateSetting(key, value)
}

func (m *Module) settingsDefs() []contracts.SettingDef {
	m.mu.Lock()
	defer m.mu.Unlock()
	return []contracts.SettingDef{
		{
			Key:         "rate",
			Label:       "Tokens Per Second",
			Type:        contracts.SettingTypeInt,
			Value:       strconv.Itoa(int(m.rate)),
			Default:     "100",
			Description: "Refill rate in tokens/sec (RATELIMIT_RATE)",
			Group:       "Limiter",
		},
		{
			Key:         "burst",
			Label:       "Burst Size",
			Type:        contracts.SettingTypeInt,
			Value:       strconv.Itoa(m.burst),
			Default:     "200",
			Description: "Maximum bucket size (RATELIMIT_BURST)",
			Group:       "Limiter",
		},
		{
			Key:         "enabled",
			Label:       "Enabled",
			Type:        contracts.SettingTypeBool,
			Value:       strconv.FormatBool(m.enabled),
			Default:     "false",
			Description: "When false, Allow always succeeds (RATELIMIT_ENABLED)",
			Group:       "Limiter",
		},
		{
			Key:         "idle_ttl",
			Label:       "Idle Bucket TTL",
			Type:        contracts.SettingTypeString,
			Value:       m.idleTTL.String(),
			Default:     defaultIdleTTL.String(),
			Description: "Drop per-key buckets after this idle period (RATELIMIT_IDLE_TTL)",
			Group:       "Limiter",
		},
	}
}

func (m *Module) updateSetting(key, value string) error {
	value = strings.TrimSpace(value)
	switch key {
	case "rate", "RATELIMIT_RATE":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil || f <= 0 {
			return fmt.Errorf("invalid rate %q (float > 0)", value)
		}
		m.mu.Lock()
		m.rate = f
		m.mu.Unlock()
		return nil
	case "burst", "RATELIMIT_BURST":
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return fmt.Errorf("invalid burst %q (integer >= 1)", value)
		}
		m.mu.Lock()
		m.burst = n
		m.mu.Unlock()
		return nil
	case "enabled", "RATELIMIT_ENABLED":
		switch strings.ToLower(value) {
		case "true", "1", "yes", "on":
			m.mu.Lock()
			m.enabled = true
			m.mu.Unlock()
		case "false", "0", "no", "off":
			m.mu.Lock()
			m.enabled = false
			m.mu.Unlock()
		default:
			return fmt.Errorf("invalid enabled %q (true/false)", value)
		}
		return nil
	case "idle_ttl", "RATELIMIT_IDLE_TTL":
		d, err := time.ParseDuration(value)
		if err != nil || d <= 0 {
			return fmt.Errorf("invalid idle_ttl %q (duration > 0)", value)
		}
		m.mu.Lock()
		m.idleTTL = d
		m.mu.Unlock()
		return nil
	default:
		return fmt.Errorf("unknown setting %q", key)
	}
}
