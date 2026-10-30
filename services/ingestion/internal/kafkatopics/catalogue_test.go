package kafkatopics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validCatalogue = `
apiVersion: observa.io/v1
kind: KafkaTopicCatalogue
defaults:
  cleanupPolicy: delete
  compressionType: producer
  maxMessageBytes: 1048576
environments:
  local:
    replicationFactor: 1
    minInsyncReplicas: 1
    topics:
      - name: observa.events.v1
        partitions: 1
        retentionMs: 259200000
        retentionBytes: 53687091200
`

func writeCatalogue(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "topics.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadCatalogue(t *testing.T) {
	path := writeCatalogue(t, validCatalogue)

	topics, err := LoadCatalogue(path, "local")
	if err != nil {
		t.Fatalf("LoadCatalogue() error = %v", err)
	}
	if len(topics) != 1 {
		t.Fatalf("topic count = %d, want 1", len(topics))
	}

	got := topics[0]
	if got.Name != "observa.events.v1" {
		t.Errorf("topic name = %q", got.Name)
	}
	if got.Partitions != 1 {
		t.Errorf("partitions = %d, want 1", got.Partitions)
	}
	if got.ReplicationFactor != 1 {
		t.Errorf("replication factor = %d, want 1", got.ReplicationFactor)
	}
	if got.Configs["retention.ms"] != "259200000" {
		t.Errorf("unexpected retention.ms = %q", got.Configs["retention.ms"])
	}
	if got.Configs["min.insync.replicas"] != "1" {
		t.Errorf("unexpected min.insync.replicas = %q", got.Configs["min.insync.replicas"])
	}
}

func TestLoadCatalogueRejectsUnknownEnvironment(t *testing.T) {
	path := writeCatalogue(t, validCatalogue)

	_, err := LoadCatalogue(path, "production")
	if err == nil {
		t.Fatal("expected unknown environment error")
	}
}

func TestLoadCatalogueRejectsDuplicateTopics(t *testing.T) {
	input := strings.Replace(
		validCatalogue,
		"        retentionBytes: 53687091200\n",
		"        retentionBytes: 53687091200\n"+
			"      - name: observa.events.v1\n"+
			"        partitions: 1\n"+
			"        retentionMs: 259200000\n"+
			"        retentionBytes: 53687091200\n",
		1,
	)

	path := writeCatalogue(t, input)
	_, err := LoadCatalogue(path, "local")
	if err == nil {
		t.Fatal("expected duplicate topic error")
	}
}

func TestLoadCatalogueRejectsInvalidPartitions(t *testing.T) {
	input := strings.Replace(
		validCatalogue,
		"partitions: 1",
		"partitions: 0",
		1,
	)
	path := writeCatalogue(t, input)

	_, err := LoadCatalogue(path, "local")
	if err == nil {
		t.Fatal("expected invalid partitions error")
	}
}

func TestLoadCatalogueRejectsMinISRAboveReplicationFactor(t *testing.T) {
	input := strings.Replace(
		validCatalogue,
		"minInsyncReplicas: 1",
		"minInsyncReplicas: 2",
		1,
	)
	path := writeCatalogue(t, input)

	_, err := LoadCatalogue(path, "local")
	if err == nil {
		t.Fatal("expected invalid min ISR error")
	}
}
