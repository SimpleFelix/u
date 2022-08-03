package u

import (
	"fmt"
	"sync"

	"github.com/Shopify/sarama"
)

type ConsumerState int

const (
	ConsumerStateReady = iota
	ConsumerStateConsuming
	ConsumerStateCancelling
	ConsumerStateRequireRestart
)

var allConsumers = make(map[string]*CommonConsumer)
var allConsumersMutex sync.RWMutex

func ConsumerForTopic(topic string) *CommonConsumer {
	allConsumersMutex.RLock()
	defer allConsumersMutex.RUnlock()
	return allConsumers[topic]
}

func saveConsumer(o *CommonConsumer) {
	allConsumersMutex.Lock()
	defer allConsumersMutex.Unlock()
	allConsumers[o.topic] = o
}

func deleteConsumer(o *CommonConsumer) {
	allConsumersMutex.Lock()
	defer allConsumersMutex.Unlock()
	delete(allConsumers, o.topic)
}

type Consumer interface {
	LastOffset() int64
	Handle(message *sarama.ConsumerMessage)
}

type CommonConsumer struct {
	canc        chan struct{}
	cancelMutex sync.Mutex
	state       ConsumerState
	topic       string
	sc          sarama.Consumer
	*CTX
	Consumer
}

func NewConsumer(topic string, sc sarama.Consumer) CommonConsumer {
	return CommonConsumer{
		topic: topic,
		sc:    sc,
		CTX:   NewContext(),
	}
}

func (r *CommonConsumer) State() ConsumerState {
	return r.state
}

func (r *CommonConsumer) changeState(state ConsumerState) {
	if state != ConsumerStateCancelling && state != ConsumerStateRequireRestart {
		return
	}
	r.cancelMutex.Lock()
	defer r.cancelMutex.Unlock()
	if r.canc != nil {
		r.state = state
		r.canc <- struct{}{}
		close(r.canc)
		r.canc = nil
	}
}

func (r *CommonConsumer) Topic() string {
	return r.topic
}

func (r *CommonConsumer) Restart() {
	r.changeState(ConsumerStateRequireRestart)
}

func (r *CommonConsumer) Cancel() {
	r.changeState(ConsumerStateCancelling)
}

func (r *CommonConsumer) MustStartConsuming() {
	if err := r.StartConsuming(); err != nil {
		panic(ErrConsumerError(err))
	}
}

func (r *CommonConsumer) StartConsuming() error {
	if r.state != ConsumerStateReady {
		msg := fmt.Sprintf("Try to start a consumer which is not ready. state=%d, topic=%v", r.state, r.topic)
		Error(msg)
		return ErrCantStartConsumer(msg)
	}
	if r.canc == nil {
		// 调用场景不存在竞争，因此没加锁。
		r.canc = make(chan struct{}, 1)
	}
	cancel := r.canc

	offset := r.LastOffset() + 1

	partitionConsumer, err := r.sc.ConsumePartition(r.topic, 0, offset)
	if err != nil {
		Errorf("Failed to create partitionConsumer topic=%v; error=%v", r.topic, err)
		return err
	}
	r.state = ConsumerStateConsuming
	Infof("Subscribed %v", r.topic)

	saveConsumer(r)
	go func() {
		defer func() {
			Errorf("closing partitionConsumer topic=%v;", r.topic)
			if err := partitionConsumer.Close(); err != nil {
				Errorf("Failed to close partitionConsumer topic=%v; err=%v", r.topic, err)
			}
			deleteConsumer(r)
			if r.state == ConsumerStateRequireRestart {
				r.state = ConsumerStateReady
				r.MustStartConsuming()
			}
		}()

	Loop:
		for {
			select {
			case msg := <-partitionConsumer.Messages():
				if r.state != ConsumerStateConsuming {
					continue
				}
				r.Handle(msg)
			case <-cancel:
				// Cancel() is invoked
				break Loop
			}
		}
	}()
	return nil
}
