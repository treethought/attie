package at

import (
	"context"
	"log/slog"

	jetstream "github.com/bluesky-social/jetstream/pkg/client"
	"github.com/bluesky-social/jetstream/pkg/client/schedulers/sequential"
	"github.com/bluesky-social/jetstream/pkg/models"
)

type JetStreamClient struct {
	sched jetstream.Scheduler
	log   *slog.Logger
	out   chan *models.Event
	err	 chan error
}

func NewJetstreamClient() *JetStreamClient {
	log := slog.Default()
	c := &JetStreamClient{
		log:   log,
		out:   make(chan *models.Event, 512),
		err: make(chan error, 1),
	}
	scheduler := sequential.NewScheduler("jetstream", slog.Default(), c.handleEvent)
	c.sched = scheduler
	return c

}

func (c *JetStreamClient) Start(ctx context.Context, cxs, dids []string, cursor *int64) {
	config := &jetstream.ClientConfig{
		WebsocketURL:      "wss://jetstream1.us-west.bsky.network/subscribe",
		Compress:          false,
		WantedDids:        dids,
		WantedCollections: cxs,
		ExtraHeaders: map[string]string{
			"User-Agent": "attie/0.0.1",
		},
	}
	jc, err := jetstream.NewClient(config, c.log, c.sched)
	if err != nil {
		c.err <- err
		return 
	}

	c.err <- jc.ConnectAndRead(ctx, cursor)
}

func (c *JetStreamClient) Out() <-chan *models.Event {
	return c.out
}
func (c *JetStreamClient) Err() <-chan error {
	return c.err
}

func (c *JetStreamClient) handleEvent(ctx context.Context, ev *models.Event) error {
	slog.Info("Received event", "did", ev.Did, "kind", ev.Kind)
	if ev.Commit == nil {
		slog.Info("skipping non commit event ", "did", ev.Did, "kind", ev.Kind)
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.out <- ev:
		return nil
	default:
		slog.Warn("deopped event", "did", ev.Did, "kind", ev.Kind)
	}
	return nil
}
