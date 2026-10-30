package kafkatopics

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Catalogue struct {
	APIVersion   string                       `yaml:"apiVersion"`
	Kind         string                       `yaml:"kind"`
	Defaults     TopicDefaults                `yaml:"defaults"`
	Environments map[string]EnvironmentConfig `yaml:"environments"`
}

type TopicDefaults struct {
	CleanupPolicy   string `yaml:"cleanupPolicy"`
	CompressionType string `yaml:"compressionType"`
	MaxMessageBytes int    `yaml:"maxMessageBytes"`
}

type EnvironmentConfig struct {
	ReplicationFactor int16         `yaml:"replicationFactor"`
	MinInsyncReplicas int16         `yaml:"minInsyncReplicas"`
	Topics            []TopicConfig `yaml:"topics"`
}

type TopicConfig struct {
	Name           string `yaml:"name"`
	Partitions     int32  `yaml:"partitions"`
	RetentionMs    int64  `yaml:"retentionMs"`
	RetentionBytes int64  `yaml:"retentionBytes"`
}

type DesiredTopic struct {
	Name              string
	Partitions        int32
	ReplicationFactor int16
	MinInsyncReplicas int16
	Configs           map[string]string
}

var validTopicName = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,249}$`)

func LoadCatalogue(path, environment string) ([]DesiredTopic, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read topic catalogue %q: %w", path, err)
	}

	var catalogue Catalogue
	if err := yaml.Unmarshal(data, &catalogue); err != nil {
		return nil, fmt.Errorf("parse topic catalogue: %w", err)
	}

	if catalogue.APIVersion != "observa.io/v1" {
		return nil, fmt.Errorf("unsupported catalogue apiVersion %q", catalogue.APIVersion)
	}
	if catalogue.Kind != "KafkaTopicCatalogue" {
		return nil, fmt.Errorf("unsupported catalogue kind %q", catalogue.Kind)
	}

	if strings.TrimSpace(environment) == "" {
		return nil, fmt.Errorf("environment is required")
	}

	env, ok := catalogue.Environments[environment]
	if !ok {
		return nil, fmt.Errorf("environment %q is not declared in the catalogue", environment)
	}

	if catalogue.Defaults.CleanupPolicy != "delete" {
		return nil, fmt.Errorf("only cleanupPolicy=delete is currently supported")
	}
	if catalogue.Defaults.CompressionType != "producer" &&
		catalogue.Defaults.CompressionType != "lz4" &&
		catalogue.Defaults.CompressionType != "zstd" &&
		catalogue.Defaults.CompressionType != "snappy" &&
		catalogue.Defaults.CompressionType != "gzip" {
		return nil, fmt.Errorf("unsupported compressionType %q", catalogue.Defaults.CompressionType)
	}
	if catalogue.Defaults.MaxMessageBytes <= 0 {
		return nil, fmt.Errorf("defaults.maxMessageBytes must be positive")
	}
	if env.ReplicationFactor < 1 {
		return nil, fmt.Errorf("replicationFactor must be at least 1")
	}
	if env.MinInsyncReplicas < 1 ||
		env.MinInsyncReplicas > env.ReplicationFactor {
		return nil, fmt.Errorf(
			"minInsyncReplicas must be between 1 and replicationFactor",
		)
	}
	if len(env.Topics) == 0 {
		return nil, fmt.Errorf("environment %q has no topics", environment)
	}

	seen := make(map[string]struct{}, len(env.Topics))
	result := make([]DesiredTopic, 0, len(env.Topics))

	for _, topic := range env.Topics {
		if !validTopicName.MatchString(topic.Name) {
			return nil, fmt.Errorf("invalid Kafka topic name %q", topic.Name)
		}
		if _, exists := seen[topic.Name]; exists {
			return nil, fmt.Errorf("duplicate topic %q", topic.Name)
		}
		seen[topic.Name] = struct{}{}

		if topic.Partitions < 1 {
			return nil, fmt.Errorf("%s: partitions must be positive", topic.Name)
		}
		if topic.RetentionMs <= 0 {
			return nil, fmt.Errorf("%s: retentionMs must be positive", topic.Name)
		}
		if topic.RetentionBytes <= 0 {
			return nil, fmt.Errorf("%s: retentionBytes must be positive", topic.Name)
		}

		result = append(result, DesiredTopic{
			Name:              topic.Name,
			Partitions:        topic.Partitions,
			ReplicationFactor: env.ReplicationFactor,
			MinInsyncReplicas: env.MinInsyncReplicas,
			Configs: map[string]string{
				"cleanup.policy":      catalogue.Defaults.CleanupPolicy,
				"compression.type":    catalogue.Defaults.CompressionType,
				"max.message.bytes":   fmt.Sprint(catalogue.Defaults.MaxMessageBytes),
				"min.insync.replicas": fmt.Sprint(env.MinInsyncReplicas),
				"retention.ms":        fmt.Sprint(topic.RetentionMs),
				"retention.bytes":     fmt.Sprint(topic.RetentionBytes),
			},
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}
