package config

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"pr-size-labeler/internal/labels"
)

type LabelsConfig struct {
	Backfill BackfillConfig
	Labels   labels.Set
}

type BackfillConfig struct {
	Enabled  bool
	Lookback time.Duration
}

type labelOverride struct {
	Name    *string `yaml:"name"`
	Lines   *int    `yaml:"lines"`
	Symbols *int    `yaml:"symbols"`
	Color   *string `yaml:"color"`
	Comment *string `yaml:"comment"`
}

const defaultBackfillLookback = 30 * 24 * time.Hour

func LoadLabelsConfig(content string) (LabelsConfig, error) {
	cfg, _, err := LoadLabelsConfigDetailed(content)
	return cfg, err
}

func LoadLabelsConfigDetailed(content string) (LabelsConfig, []string, error) {
	if strings.TrimSpace(content) == "" {
		return LabelsConfig{
			Backfill: BackfillConfig{Lookback: defaultBackfillLookback},
			Labels:   labels.DefaultSet().Clone(),
		}, nil, nil
	}

	decoder := yaml.NewDecoder(bytes.NewBufferString(content))
	var root yaml.Node
	if err := decoder.Decode(&root); err != nil {
		return LabelsConfig{}, nil, err
	}
	warnings := collectLabelsConfigWarnings(&root)
	rootMapping, err := labelsConfigRootMapping(&root)
	if err != nil {
		return LabelsConfig{}, warnings, err
	}

	resolvedBackfill, err := resolveBackfillConfig(rootMapping["backfill"])
	if err != nil {
		return LabelsConfig{}, warnings, err
	}
	resolvedLabels, err := resolveLabelSet(rootMapping["labels"])
	if err != nil {
		return LabelsConfig{}, warnings, err
	}

	return LabelsConfig{
		Backfill: resolvedBackfill,
		Labels:   resolvedLabels,
	}, warnings, nil
}

func resolveBackfillConfig(node *yaml.Node) (BackfillConfig, error) {
	cfg := BackfillConfig{Lookback: defaultBackfillLookback}
	if node == nil {
		return cfg, nil
	}
	if node.Kind != yaml.MappingNode {
		return BackfillConfig{}, errors.New("backfill must be a mapping")
	}

	for i := 0; i+1 < len(node.Content); i += 2 {
		key := strings.TrimSpace(node.Content[i].Value)
		value := node.Content[i+1]
		switch key {
		case "enabled":
			var enabled bool
			if err := value.Decode(&enabled); err != nil {
				return BackfillConfig{}, err
			}
			cfg.Enabled = enabled
		case "lookback":
			var lookback string
			if err := value.Decode(&lookback); err != nil {
				return BackfillConfig{}, err
			}
			lookback = strings.TrimSpace(lookback)
			if lookback == "" {
				continue
			}
			duration, err := time.ParseDuration(lookback)
			if err != nil {
				return BackfillConfig{}, fmt.Errorf("parse backfill.lookback: %w", err)
			}
			if duration <= 0 {
				return BackfillConfig{}, errors.New("backfill.lookback must be greater than 0")
			}
			cfg.Lookback = duration
		}
	}
	return cfg, nil
}

func resolveLabelSet(node *yaml.Node) (labels.Set, error) {
	set := labels.DefaultSet().Clone()
	if node == nil {
		return set, nil
	}
	if node.Kind != yaml.MappingNode {
		return nil, errors.New("labels must be a mapping")
	}

	for i := 0; i+1 < len(node.Content); i += 2 {
		key := strings.TrimSpace(node.Content[i].Value)
		overrideNode := node.Content[i+1]
		def, ok := set[key]
		if !ok {
			continue
		}
		override, err := parseLabelOverride(key, overrideNode)
		if err != nil {
			return nil, err
		}
		merged := def
		if override.Name != nil {
			merged.Name = *override.Name
		}
		if override.Lines != nil {
			merged.Lines = *override.Lines
		}
		if override.Symbols != nil {
			symbols := *override.Symbols
			merged.Symbols = &symbols
		}
		if override.Color != nil {
			merged.Color = *override.Color
		}
		if override.Comment != nil {
			merged.Comment = *override.Comment
		}
		set[key] = merged
	}

	return set, nil
}

func labelsConfigRootMapping(root *yaml.Node) (map[string]*yaml.Node, error) {
	node := root
	if node == nil {
		return nil, nil
	}
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return nil, nil
		}
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return nil, errors.New("labels config must be a mapping")
	}

	mapping := make(map[string]*yaml.Node, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		mapping[strings.TrimSpace(node.Content[i].Value)] = node.Content[i+1]
	}
	return mapping, nil
}

func parseLabelOverride(labelKey string, node *yaml.Node) (labelOverride, error) {
	if node == nil {
		return labelOverride{}, nil
	}
	if node.Kind != yaml.MappingNode {
		return labelOverride{}, fmt.Errorf("labels.%s must be a mapping", labelKey)
	}

	var override labelOverride
	for i := 0; i+1 < len(node.Content); i += 2 {
		field := strings.TrimSpace(node.Content[i].Value)
		value := node.Content[i+1]
		switch field {
		case "name":
			var parsed string
			if err := value.Decode(&parsed); err != nil {
				return labelOverride{}, err
			}
			override.Name = &parsed
		case "lines":
			var parsed int
			if err := value.Decode(&parsed); err != nil {
				return labelOverride{}, err
			}
			override.Lines = &parsed
		case "symbols":
			var parsed int
			if err := value.Decode(&parsed); err != nil {
				return labelOverride{}, err
			}
			override.Symbols = &parsed
		case "color":
			var parsed string
			if err := value.Decode(&parsed); err != nil {
				return labelOverride{}, err
			}
			override.Color = &parsed
		case "comment":
			var parsed string
			if err := value.Decode(&parsed); err != nil {
				return labelOverride{}, err
			}
			override.Comment = &parsed
		}
	}
	return override, nil
}

func collectLabelsConfigWarnings(root *yaml.Node) []string {
	node := root
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return nil
		}
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}

	warnings := []string{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := strings.TrimSpace(node.Content[i].Value)
		value := node.Content[i+1]
		switch key {
		case "backfill":
			warnings = append(warnings, collectBackfillWarnings(value)...)
		case "labels":
			warnings = append(warnings, collectLabelWarnings(value)...)
		default:
			warnings = append(warnings, fmt.Sprintf("unsupported top-level key %q ignored", key))
		}
	}
	return warnings
}

func collectBackfillWarnings(node *yaml.Node) []string {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}

	warnings := []string{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := strings.TrimSpace(node.Content[i].Value)
		switch key {
		case "enabled", "lookback":
		default:
			warnings = append(warnings, fmt.Sprintf("unsupported key %q ignored", "backfill."+key))
		}
	}
	return warnings
}

func collectLabelWarnings(node *yaml.Node) []string {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}

	defaults := labels.DefaultSet()
	warnings := []string{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		labelKey := strings.TrimSpace(node.Content[i].Value)
		override := node.Content[i+1]
		if _, ok := defaults[labelKey]; !ok {
			warnings = append(warnings, fmt.Sprintf("unsupported labels key %q ignored", "labels."+labelKey))
			continue
		}
		if override.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j+1 < len(override.Content); j += 2 {
			field := strings.TrimSpace(override.Content[j].Value)
			switch field {
			case "name", "lines", "symbols", "color", "comment":
			default:
				warnings = append(warnings, fmt.Sprintf("unsupported key %q ignored", "labels."+labelKey+"."+field))
			}
		}
	}
	return warnings
}
