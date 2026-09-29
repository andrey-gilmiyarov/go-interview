package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/config"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/event"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

var ErrDeliveryOutcomeUnknown = errors.New("Kafka delivery outcome unknown")

type deliveryResult struct {
	record *kgo.Record
	err    error
}

func baseOptions(cfg config.Config) []kgo.Opt {
	return []kgo.Opt{kgo.SeedBrokers(cfg.Brokers...), kgo.ClientID("go-interview-kafka-lab")}
}

// #region new-producer

func NewProducer(cfg config.Config) (*kgo.Client, error) {
	opts := append(baseOptions(cfg), kgo.RequiredAcks(kgo.AllISRAcks()))
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("create Kafka producer: %w", err)
	}
	return client, nil
}

// #endregion new-producer

// #region new-consumer

// NewConsumer creates a classic-protocol group consumer with automatic commits disabled.
func NewConsumer(cfg config.Config, topic, group string, extra ...kgo.Opt) (*kgo.Client, error) {
	if topic == "" || group == "" {
		return nil, fmt.Errorf("consumer topic and group are required")
	}
	opts := append(baseOptions(cfg), extra...)
	opts = append(opts, kgo.ConsumerGroup(group), kgo.ConsumeTopics(topic), kgo.ConsumeStartOffset(kgo.NewOffset().AtStart()), kgo.DisableAutoCommit(), kgo.FetchMaxWait(200*time.Millisecond))
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("create Kafka consumer for topic %q group %q: %w", topic, group, err)
	}
	return client, nil
}

// #endregion new-consumer

func NewTransactionalGroup(cfg config.Config, topic, group, transactionalID string) (*kgo.GroupTransactSession, error) {
	if topic == "" || group == "" || transactionalID == "" {
		return nil, fmt.Errorf("transaction topic, group, and transactional ID are required")
	}
	opts := append(baseOptions(cfg), kgo.ConsumerGroup(group), kgo.ConsumeTopics(topic), kgo.ConsumeStartOffset(kgo.NewOffset().AtStart()), kgo.TransactionalID(transactionalID), kgo.FetchIsolationLevel(kgo.ReadCommitted()), kgo.DisableAutoCommit())
	session, err := kgo.NewGroupTransactSession(opts...)
	if err != nil {
		return nil, fmt.Errorf("create transactional Kafka group: %w", err)
	}
	return session, nil
}

// #region produce-event

func ProduceEvent(ctx context.Context, client *kgo.Client, topic string, e event.Event) (*kgo.Record, error) {
	body, err := event.Encode(e)
	if err != nil {
		return nil, err
	}
	if client == nil || topic == "" {
		return nil, fmt.Errorf("Kafka client and topic are required")
	}
	record := &kgo.Record{Topic: topic, Key: []byte(e.OrderID), Value: body}
	delivered := make(chan deliveryResult, 1)
	client.Produce(ctx, record, func(record *kgo.Record, err error) {
		delivered <- deliveryResult{record: record, err: err}
	})
	return awaitDelivery(ctx, delivered, e.EventID, topic)
}

func awaitDelivery(ctx context.Context, delivered <-chan deliveryResult, eventID, topic string) (*kgo.Record, error) {
	resultError := func(result deliveryResult) (*kgo.Record, error) {
		if result.err != nil {
			return nil, fmt.Errorf("produce event %q to topic %q: %w", eventID, topic, result.err)
		}
		return result.record, nil
	}
	select {
	case result := <-delivered:
		return resultError(result)
	case <-ctx.Done():
		// Select may choose ctx.Done even after a confirmed callback filled the
		// buffered channel. Prefer that known result before reporting uncertainty.
		select {
		case result := <-delivered:
			return resultError(result)
		default:
		}
		// Idempotent in-flight records may outlive this wait. Retire this client
		// after an unknown outcome; do not retry with it or claim the send failed.
		return nil, fmt.Errorf("%w for event %q to topic %q: %v", ErrDeliveryOutcomeUnknown, eventID, topic, ctx.Err())
	}
}

// #endregion produce-event

func DecodeRecord(record *kgo.Record) (event.Event, error) {
	if record == nil {
		return event.Event{}, fmt.Errorf("Kafka record is nil")
	}
	e, err := event.Decode(record.Value)
	if err != nil {
		return event.Event{}, fmt.Errorf("decode topic %q partition %d offset %d: %w", record.Topic, record.Partition, record.Offset, err)
	}
	if string(record.Key) != e.OrderID {
		return event.Event{}, fmt.Errorf("record key %q does not match event order_id %q", record.Key, e.OrderID)
	}
	return e, nil
}

func CreateTopic(ctx context.Context, cfg config.Config, topic string, partitions int32, minISR int16) error {
	request, err := createTopicsRequest(topic, partitions, cfg.ReplicationFactor, minISR)
	if err != nil {
		return err
	}
	client, err := kgo.NewClient(baseOptions(cfg)...)
	if err != nil {
		return fmt.Errorf("create Kafka topic admin client: %w", err)
	}
	defer client.Close()
	response, err := request.RequestWith(ctx, client)
	if err != nil {
		return fmt.Errorf("create Kafka topic %q: %w", topic, err)
	}
	for _, result := range response.Topics {
		resultErr := kerr.ErrorForCode(result.ErrorCode)
		if resultErr != nil && !errors.Is(resultErr, kerr.TopicAlreadyExists) {
			return fmt.Errorf("create Kafka topic %q: %w", result.Topic, resultErr)
		}
	}
	return nil
}

func DeleteTopic(ctx context.Context, cfg config.Config, topic string) error {
	request, err := deleteTopicsRequest(topic)
	if err != nil {
		return err
	}
	client, err := kgo.NewClient(baseOptions(cfg)...)
	if err != nil {
		return fmt.Errorf("create Kafka topic admin client: %w", err)
	}
	defer client.Close()
	response, err := request.RequestWith(ctx, client)
	if err != nil {
		return fmt.Errorf("delete Kafka topic %q: %w", topic, err)
	}
	if len(response.Topics) != 1 || response.Topics[0].Topic == nil || *response.Topics[0].Topic != topic {
		return fmt.Errorf("delete Kafka topic %q: response did not identify the requested topic", topic)
	}
	deleteErr := kerr.ErrorForCode(response.Topics[0].ErrorCode)
	if deleteErr != nil && !errors.Is(deleteErr, kerr.UnknownTopicOrPartition) {
		return fmt.Errorf("delete Kafka topic %q: %w", topic, deleteErr)
	}
	return nil
}

func deleteTopicsRequest(topic string) (*kmsg.DeleteTopicsRequest, error) {
	if topic == "" {
		return nil, fmt.Errorf("topic name is required")
	}
	request := kmsg.NewPtrDeleteTopicsRequest()
	request.TimeoutMillis = 10_000
	request.TopicNames = []string{topic}
	topicByName := kmsg.NewDeleteTopicsRequestTopic()
	topicByName.Topic = &topic
	request.Topics = []kmsg.DeleteTopicsRequestTopic{topicByName}
	return request, nil
}

func DeleteConsumerGroup(ctx context.Context, cfg config.Config, group string) error {
	request, err := deleteGroupsRequest(group)
	if err != nil {
		return err
	}
	client, err := kgo.NewClient(baseOptions(cfg)...)
	if err != nil {
		return fmt.Errorf("create Kafka group admin client: %w", err)
	}
	defer client.Close()
	response, err := request.RequestWith(ctx, client)
	if err != nil {
		return fmt.Errorf("delete Kafka consumer group %q: %w", group, err)
	}
	if len(response.Groups) != 1 || response.Groups[0].Group != group {
		return fmt.Errorf("delete Kafka consumer group %q: response did not identify the requested group", group)
	}
	deleteErr := kerr.ErrorForCode(response.Groups[0].ErrorCode)
	if deleteErr != nil && !errors.Is(deleteErr, kerr.GroupIDNotFound) {
		return fmt.Errorf("delete Kafka consumer group %q: %w", group, deleteErr)
	}
	return nil
}

func deleteGroupsRequest(group string) (*kmsg.DeleteGroupsRequest, error) {
	if group == "" {
		return nil, fmt.Errorf("group ID is required")
	}
	request := kmsg.NewPtrDeleteGroupsRequest()
	request.Groups = []string{group}
	return request, nil
}

func createTopicsRequest(topic string, partitions int32, replicationFactor, minISR int16) (*kmsg.CreateTopicsRequest, error) {
	if topic == "" {
		return nil, fmt.Errorf("topic name is required")
	}
	if partitions < 1 {
		return nil, fmt.Errorf("topic partitions must be positive")
	}
	if replicationFactor < 1 {
		return nil, fmt.Errorf("topic replication factor must be positive")
	}
	if minISR < 0 || minISR > replicationFactor {
		return nil, fmt.Errorf("minimum in-sync replicas must be between zero and replication factor")
	}
	request := kmsg.NewPtrCreateTopicsRequest()
	request.TimeoutMillis = 10_000
	requestTopic := kmsg.NewCreateTopicsRequestTopic()
	requestTopic.Topic = topic
	requestTopic.NumPartitions = partitions
	requestTopic.ReplicationFactor = replicationFactor
	if minISR > 0 {
		value := fmt.Sprintf("%d", minISR)
		requestTopic.Configs = append(requestTopic.Configs, kmsg.CreateTopicsRequestTopicConfig{Name: "min.insync.replicas", Value: &value})
	}
	request.Topics = []kmsg.CreateTopicsRequestTopic{requestTopic}
	return request, nil
}
