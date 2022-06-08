package worker

import (
	"fmt"
	"raccoon/logger"
	"raccoon/metrics"
	ws "raccoon/websocket"
	"sync"
	"time"

	"raccoon/publisher"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

// Pool spawn goroutine as much as Size that will listen to EventsChannel. On Close, wait for all data in EventsChannel to be processed.
type Pool struct {
	Size                int
	deliveryChannelSize int
	EventsChannel       <-chan ws.EventsBatch
	kafkaProducer       publisher.KafkaProducer
	wg                  sync.WaitGroup
}

// CreateWorkerPool create new Pool struct given size and EventsChannel worker.
func CreateWorkerPool(size int, eventsChannel <-chan ws.EventsBatch, deliveryChannelSize int, kafkaProducer publisher.KafkaProducer) *Pool {
	return &Pool{
		Size:                size,
		deliveryChannelSize: deliveryChannelSize,
		EventsChannel:       eventsChannel,
		kafkaProducer:       kafkaProducer,
		wg:                  sync.WaitGroup{},
	}
}

// StartWorkers initialize worker pool as much as Pool.Size
func (w *Pool) StartWorkers() {
	w.wg.Add(w.Size)
	for i := 0; i < w.Size; i++ {
		go func(workerName string) {
			logger.Info("Running worker: " + workerName)
			deliveryChan := make(chan kafka.Event, w.deliveryChannelSize)
			go func() {
				for d := range deliveryChan {
					m := d.(*kafka.Message)
					if m.TopicPartition.Error != nil {
						logger.Errorf("[worker] Fail to publish message to kafka %v", m.TopicPartition.Error)
						metrics.Increment("kafka_messages_delivered_total", fmt.Sprintf("success=false,conn_group=%s", ""))
					}
				}
			}()
			for request := range w.EventsChannel {
				metrics.Timing("batch_idle_in_channel_milliseconds", (time.Now().Sub(request.TimePushed)).Milliseconds(), "worker="+workerName)
				batchReadTime := time.Now()
				//@TODO - Should add integration tests to prove that the worker receives the same message that it produced, on the delivery channel it created

				err := w.kafkaProducer.ProduceBulk(request.EventReq.GetEvents(), deliveryChan)

				totalErr := 0
				if err != nil {
					for _, err := range err.(publisher.BulkError).Errors {
						if err != nil {
							logger.Errorf("[worker] Fail to publish message to kafka %v", err)
							totalErr++
						}
					}
				}
				lenBatch := int64(len(request.EventReq.GetEvents()))
				logger.Debug(fmt.Sprintf("Success sending messages, %v", lenBatch-int64(totalErr)))
				if lenBatch > 0 {
					eventTimingMs := time.Since(time.Unix(request.EventReq.SentTime.Seconds, 0)).Milliseconds() / lenBatch
					metrics.Timing("event_processing_duration_milliseconds", eventTimingMs, fmt.Sprintf("conn_group=%s", request.ConnIdentifier.Group))
					now := time.Now()
					metrics.Timing("worker_processing_duration_milliseconds", (now.Sub(batchReadTime).Milliseconds())/lenBatch, "worker="+workerName)
					metrics.Timing("server_processing_latency_milliseconds", (now.Sub(request.TimeConsumed)).Milliseconds()/lenBatch, fmt.Sprintf("conn_group=%s", request.ConnIdentifier.Group))
				}
				metrics.Count("kafka_messages_delivered_total", totalErr, fmt.Sprintf("success=false,conn_group=%s", request.ConnIdentifier.Group))
				metrics.Count("kafka_messages_delivered_total", len(request.EventReq.GetEvents())-totalErr, fmt.Sprintf("success=true,conn_group=%s", request.ConnIdentifier.Group))
			}
			w.wg.Done()
		}(fmt.Sprintf("worker-%d", i))
	}
}

// FlushWithTimeOut waits for the workers to complete the pending the messages
//to be flushed to the publisher within a timeout.
// Returns true if waiting timed out, meaning not all the events could be processed before this timeout.
func (w *Pool) FlushWithTimeOut(timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		w.wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
