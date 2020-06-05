package publisher

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/mock"
)

type MockKafkaProducer struct {
	mock.Mock
}

func (p *MockKafkaProducer) Produce(m *kafka.Message, eventsChan chan kafka.Event) error {
	args := p.Called(m, eventsChan)
	return args.Error(0)
}

func (p *MockKafkaProducer) Len() int {
	args := p.Called()
	return args.Int(0)
}

func (p *MockKafkaProducer) Events() chan kafka.Event {
	args := p.Called()
	return args.Get(0).(chan kafka.Event)
}

func (p *MockKafkaProducer) Flush(num int) int {
	args := p.Called(num)
	return args.Int(0)
}

func (p *MockKafkaProducer) Close() {
	p.Called()
}
