package p2p

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/CESSProject/p2p-go/core"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/pkg/errors"
)

func Subscribe(ctx context.Context, h host.Host, bootnode string, recv chan<- peer.AddrInfo) error {
	if recv == nil {
		err := errors.New("empty receive channel")
		return errors.Wrap(err, "subscribe peer node error")
	}
	gossip, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return errors.Wrap(err, "subscribe peer node error")
	}
	room := fmt.Sprintf("%s-%s", core.NetworkRoom, bootnode)
	s := mdns.NewMdnsService(h, "", nil)
	err = s.Start()
	if err != nil {
		errors.Wrap(err, "subscribe peer node error")
	}
	topic, err := gossip.Join(room)
	if err != nil {
		errors.Wrap(err, "subscribe peer node error")
	}
	defer topic.Close()
	subscriber, err := topic.Subscribe()
	if err != nil {
		errors.Wrap(err, "subscribe peer node error")
	}
	defer subscriber.Cancel()
	var fpeer peer.AddrInfo
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, err := subscriber.Next(ctx)
			if err != nil {
				continue
			}
			if msg.ReceivedFrom == h.ID() {
				continue
			}
			err = json.Unmarshal(msg.Data, &fpeer)
			if err != nil || fpeer.ID.Size() == 0 {
				continue
			}
			recv <- fpeer
		}
	}
}
