package actors

import (
	"github.com/CESSProject/cess-dcdn-components/cd2n-lib/p2p"
	"github.com/asynkron/protoactor-go/actor"
	"github.com/libp2p/go-libp2p/core/host"
)

// Use p2p transfer actor to implement http to p2p request

type Handle func(msg P2pMessage, host host.Host)

type P2pNodeActor struct {
	*p2p.PeerNode
	handleMap map[string]Handle
}

type P2pMessage struct {
	Option string
	Data   any
}

func NewP2pNodeActor(node *p2p.PeerNode) *P2pNodeActor {
	return &P2pNodeActor{
		PeerNode: node,
	}
}

func (peer *P2pNodeActor) RegisterMessageHandle(option string, handle Handle) {
	peer.handleMap[option] = handle
}

func (peer *P2pNodeActor) Receive(context actor.Context) {
	msg, ok := context.Message().(P2pMessage)
	if !ok {
		return
	}
	handle := peer.handleMap[msg.Option]
	if handle != nil {
		handle(msg, peer.Host)
	}
}
