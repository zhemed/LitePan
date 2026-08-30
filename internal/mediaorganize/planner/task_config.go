package planner

import (
	"fmt"
	"strings"

	"litepan/internal/mediaorganize/rules"
)

func TaskConfigFromMap(cfg map[string]any) TaskConfig {
	if cfg == nil {
		return TaskConfig{}
	}
	return TaskConfig{
		TargetDirectoryID:    strings.TrimSpace(mapString(cfg, "target_directory_id")),
		TargetRootID:         strings.TrimSpace(mapString(cfg, "target_root_id")),
		ActionType:           strings.TrimSpace(mapString(cfg, "action_type")),
		MediaType:            strings.TrimSpace(mapString(cfg, "media_type")),
		RenameMarker:         strings.TrimSpace(mapString(cfg, "rename_marker")),
		UseTMDB:              rules.SettingBool(cfg["use_tmdb"], false),
		OverwriteExisting:    rules.SettingBool(cfg["overwrite_existing"], false),
		Recursive:            rules.SettingBool(cfg["recursive"], false),
		SeasonFolderTemplate: strings.TrimSpace(mapString(cfg, "season_folder_template")),
		FileExtensions:       strings.TrimSpace(mapString(cfg, "file_extensions")),
		MetadataExtensions:   strings.TrimSpace(mapString(cfg, "metadata_extensions")),
	}
}

func mapString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
