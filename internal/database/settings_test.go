package database

import (
	"testing"

	"Postwoman/internal/model"
)

func TestMergeSettingsAppliesNewDefaults(t *testing.T) {
	got := mergeSettings(model.DefaultAppSettings(), model.AppSettings{})
	if got.FontSize != 15 {
		t.Errorf("default interface font size = %d, want 15", got.FontSize)
	}
	if got.UIFont != "system" {
		t.Errorf("default interface font = %q, want system", got.UIFont)
	}
	if got.EditorFontFamily != "mono" || got.EditorFontSize != 13 {
		t.Errorf("editor defaults changed: family=%q size=%d", got.EditorFontFamily, got.EditorFontSize)
	}
}

func TestMergeSettingsMigratesLegacyFontDefaults(t *testing.T) {
	// 13 px and 16 px were the shipped defaults in earlier releases.
	for _, legacySize := range []int{13, 16} {
		got := mergeSettings(model.DefaultAppSettings(), model.AppSettings{FontSize: legacySize, UIFont: "system"})
		if got.FontSize != 15 {
			t.Errorf("legacy size %d: got %d, want 15", legacySize, got.FontSize)
		}
		if got.UIFont != "system" {
			t.Errorf("legacy size %d: interface font = %q, want system", legacySize, got.UIFont)
		}
	}
}

func TestMergeSettingsMigratesBoldOnlyOpenSansDefault(t *testing.T) {
	// v1.3.3 briefly made a Bold-only Open Sans face the interface default.
	// Those stored values are moved back to the natural system sans.
	got := mergeSettings(model.DefaultAppSettings(), model.AppSettings{UIFont: "open-sans"})
	if got.UIFont != "system" {
		t.Errorf("interface font = %q, want system", got.UIFont)
	}
}

func TestMergeSettingsKeepsDeliberateChoices(t *testing.T) {
	got := mergeSettings(model.DefaultAppSettings(), model.AppSettings{
		Theme:                  "dark",
		FontSize:               18,
		EditorFontSize:         16,
		EditorFontFamily:       "system",
		UIFont:                 "source-code-pro",
		AccentColor:            "violet",
		CompactMode:            true,
		LayoutPreset:           "json",
		RequestTimeout:         5000,
		HistoryRetentionDays:   7,
		SendNoCacheHeader:      true,
		ScriptExecutionEnabled: false,
		KeyboardShortcuts:      map[string]string{"send": "Ctrl+Shift+Enter"},
	})
	if got.Theme != "dark" || got.FontSize != 18 || got.UIFont != "source-code-pro" {
		t.Errorf("theme/size/font not preserved: %q %d %q", got.Theme, got.FontSize, got.UIFont)
	}
	if got.EditorFontSize != 16 || got.EditorFontFamily != "system" || got.AccentColor != "violet" {
		t.Errorf("editor/accent not preserved: %d %q %q", got.EditorFontSize, got.EditorFontFamily, got.AccentColor)
	}
	if !got.CompactMode || got.LayoutPreset != "json" || got.RequestTimeout != 5000 {
		t.Errorf("layout/request settings not preserved: %+v", got)
	}
	if got.HistoryRetentionDays != 7 || got.ScriptExecutionEnabled || !got.SendNoCacheHeader {
		t.Errorf("script/history settings not preserved: %+v", got)
	}
	if got.KeyboardShortcuts["send"] != "Ctrl+Shift+Enter" {
		t.Errorf("shortcut overrides not preserved: %+v", got.KeyboardShortcuts)
	}
}
