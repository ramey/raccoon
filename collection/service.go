package collection

import (
	"context"
	"time"
)

type ChannelCollector struct {
	ch chan CollectRequest
}

func NewChannelCollector(c chan CollectRequest) *ChannelCollector {
	return &ChannelCollector{
		ch: c,
	}
}

func (c *ChannelCollector) Collect(ctx context.Context, req *CollectRequest) error {
	req.TimePushed = time.Now()
	c.ch <- *req
	return nil
}
