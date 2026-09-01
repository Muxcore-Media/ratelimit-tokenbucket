package internal

import "testing"

func TestSettingsRateBurstEnabled(t *testing.T) {
	m := NewModule(Config{GRPCAddr: "127.0.0.1:0", Rate: 10, Burst: 5, Enabled: true})
	defs := m.Settings()
	if len(defs) != 4 {
		t.Fatalf("settings=%d", len(defs))
	}
	if err := m.UpdateSetting("rate", "50"); err != nil {
		t.Fatal(err)
	}
	if err := m.UpdateSetting("burst", "25"); err != nil {
		t.Fatal(err)
	}
	if err := m.UpdateSetting("enabled", "false"); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	rate, burst, enabled := m.rate, m.burst, m.enabled
	m.mu.Unlock()
	if rate != 50 || burst != 25 || enabled {
		t.Fatalf("got rate=%v burst=%v enabled=%v", rate, burst, enabled)
	}
	if err := m.UpdateSetting("rate", "0"); err == nil {
		t.Fatal("expected error")
	}
}
