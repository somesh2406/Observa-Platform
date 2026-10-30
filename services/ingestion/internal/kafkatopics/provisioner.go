package kafkatopics

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Provisioner struct {
	admin *kadm.Client
}

func NewProvisioner(client *kgo.Client) *Provisioner {
	return &Provisioner{admin: kadm.NewClient(client)}
}

func (p *Provisioner) Close() {
	p.admin.Close()
}

func (p *Provisioner) Run(ctx context.Context, desired []DesiredTopic) error {
	if len(desired) == 0 {
		return fmt.Errorf("no topics requested for provisioning")
	}

	brokers, err := p.admin.BrokerMetadata(ctx)
	if err != nil {
		return fmt.Errorf("describe Kafka brokers: %w", err)
	}
	if len(brokers.Brokers) == 0 {
		return fmt.Errorf("Kafka returned no brokers")
	}

	// Refuse an impossible replication policy before mutating the cluster.
	for _, topic := range desired {
		if int(topic.ReplicationFactor) > len(brokers.Brokers) {
			return fmt.Errorf(
				"%s: replication factor %d exceeds available broker count %d",
				topic.Name, topic.ReplicationFactor, len(brokers.Brokers),
			)
		}
	}

	for _, topic := range desired {
		if err := p.reconcileTopic(ctx, topic); err != nil {
			return fmt.Errorf("reconcile topic %s: %w", topic.Name, err)
		}
	}

	return nil
}

func (p *Provisioner) reconcileTopic(ctx context.Context, desired DesiredTopic) error {
	topics, err := p.admin.ListTopics(ctx)
	if err != nil {
		return fmt.Errorf("list Kafka topics: %w", err)
	}

	if !topics.Has(desired.Name) {
		configs := make(map[string]*string, len(desired.Configs))
		for key, value := range desired.Configs {
			configs[key] = kadm.StringPtr(value)
		}

		response, err := p.admin.CreateTopic(
			ctx,
			desired.Partitions,
			desired.ReplicationFactor,
			configs,
			desired.Name,
		)
		if err != nil {
			return fmt.Errorf("create topic: %w", err)
		}
		if response.Err != nil {
			return fmt.Errorf("create topic: %w", response.Err)
		}
		fmt.Printf("created topic=%s partitions=%d replication=%d\n",
			desired.Name, desired.Partitions, desired.ReplicationFactor)
	} else {
		fmt.Printf("topic exists; verifying topic=%s\n", desired.Name)
	}

	// Re-read broker metadata rather than trusting the create response.
	metadata, err := p.admin.Metadata(ctx, desired.Name)
	if err != nil {
		return fmt.Errorf("read topic metadata: %w", err)
	}

	detail, ok := metadata.Topics[desired.Name]
	if !ok {
		return fmt.Errorf("topic missing from metadata after provisioning")
	}
	if detail.Err != nil {
		return fmt.Errorf("topic metadata error: %w", detail.Err)
	}

	if len(detail.Partitions) != int(desired.Partitions) {
		return fmt.Errorf(
			"partition drift: desired=%d actual=%d; partition changes require an explicit migration",
			desired.Partitions, len(detail.Partitions),
		)
	}

	partitionIDs := make([]int, 0, len(detail.Partitions))
	for id := range detail.Partitions {
		partitionIDs = append(partitionIDs, int(id))
	}
	sort.Ints(partitionIDs)

	for _, id := range partitionIDs {
		partition := detail.Partitions[int32(id)]
		if partition.Err != nil {
			return fmt.Errorf("partition %d metadata: %w", id, partition.Err)
		}
		if len(partition.Replicas) != int(desired.ReplicationFactor) {
			return fmt.Errorf(
				"partition %d replication drift: desired=%d actual=%d; replica reassignment requires an explicit migration",
				id, desired.ReplicationFactor, len(partition.Replicas),
			)
		}
	}

	if err := p.reconcileConfigs(ctx, desired); err != nil {
		return err
	}

	if err := p.verifyConfigs(ctx, desired); err != nil {
		return err
	}

	fmt.Printf("verified topic=%s partitions=%d replication=%d\n",
		desired.Name, desired.Partitions, desired.ReplicationFactor)
	return nil
}

func (p *Provisioner) reconcileConfigs(ctx context.Context, desired DesiredTopic) error {
	actual, err := p.readConfigs(ctx, desired.Name)
	if err != nil {
		return err
	}

	var changes []kadm.AlterConfig
	for key, wanted := range desired.Configs {
		if actual[key] == wanted {
			continue
		}
		changes = append(changes, kadm.AlterConfig{
			Op:    kadm.SetConfig,
			Name:  key,
			Value: kadm.StringPtr(wanted),
		})
	}

	if len(changes) == 0 {
		return nil
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Name < changes[j].Name
	})

	responses, err := p.admin.AlterTopicConfigs(ctx, changes, desired.Name)
	if err != nil {
		return fmt.Errorf("alter topic configs: %w", err)
	}
	for _, response := range responses {
		if response.Err != nil {
			return fmt.Errorf(
				"alter configs for %s: %w",
				response.Name, response.Err,
			)
		}
	}

	fmt.Printf("updated topic configuration topic=%s keys=%d\n",
		desired.Name, len(changes))
	return nil
}

func (p *Provisioner) verifyConfigs(
	ctx context.Context,
	desired DesiredTopic,
) error {
	actual, err := p.readConfigs(ctx, desired.Name)
	if err != nil {
		return err
	}

	for key, wanted := range desired.Configs {
		got, exists := actual[key]
		if !exists {
			return fmt.Errorf("configuration %q missing after reconciliation", key)
		}
		if got != wanted {
			return fmt.Errorf(
				"configuration drift for %s: desired %s=%q actual=%q",
				desired.Name, key, wanted, got,
			)
		}
	}
	return nil
}

func (p *Provisioner) readConfigs(
	ctx context.Context,
	topic string,
) (map[string]string, error) {
	resources, err := p.admin.DescribeTopicConfigs(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("describe topic configs: %w", err)
	}

	for _, resource := range resources {
		if resource.Name != topic {
			continue
		}
		if resource.Err != nil {
			return nil, fmt.Errorf("describe configs for %s: %w", topic, resource.Err)
		}

		result := make(map[string]string, len(resource.Configs))
		for _, config := range resource.Configs {
			result[config.Key] = config.MaybeValue()
		}
		return result, nil
	}

	return nil, fmt.Errorf("Kafka returned no config resource for topic %s", topic)
}

// ParseDurationSeconds is kept here for potential future provisioning timeouts;
// it is not used to change topic configuration.
func ParseDurationSeconds(value string) (time.Duration, error) {
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("invalid positive duration in seconds %q", value)
	}
	return time.Duration(seconds) * time.Second, nil
}
