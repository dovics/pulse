package message

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/nats-io/nuid"
	"github.com/silverswords/pulse/utils"
)

var uidGen = nuid.New()

const (
	// DefaultCloudEventType is the default event type for an pulse published event
	DefaultCloudEventType = "com.pulse.event.sent"
	// CloudEventsSpecVersion is the specversion used by pulse for the cloud events implementation
	CloudEventsSpecVersion = "1.0"
	//ContentType is the Cloud Events HTTP content type
	ContentType = "application/cloudevents+json"
	// DefaultCloudEventSource is the default event source
	DefaultCloudEventSource = "pulse"
)

// todo: transform those message to cloudEvent specification.
// Message format maybe below
//{
//"specversion": "1.0",
//"type": "com.pulse.event.sent",
//"id": "xxxx-xxxx-xxxx",
//"source": "pulse",
//"data": { ... }
//"datacontenttype":"application/cloudevents+json"
//}
type Message struct {
	// did we need a ack id again? no because the message id is just for pulse.
	// it's enough to ack.
	// AckID string
	Id   string
	Data []byte // Message data

	OrderingKey string // for example, order id, would be ordered consume by the consumer.
	// Where the message from and to. what codec is the message have. when and why have this message.
	Attributes utils.Header // Message Header use to specific message and how to handle it.

	// Logic is represents the fields that don't need initialize by the message producer.
	Size int
	// DeliveryAttempt is the number of times a message has been delivered.
	// This is part of the dead lettering feature that forwards messages that
	// fail to be processed (from nack/ack deadline timeout) to a dead letter topic.
	// If dead lettering is enabled, this will be set on all attempts, starting
	// with value 1. Otherwise, the value will be nil.
	// This field is read-only.
	DeliveryAttempt *int
	calledDone      bool
	DoneFunc        func(string, bool)
}

// hint: now message string just print the event
func (m *Message) String() string {
	return fmt.Sprintf("Id: %s Data: %s Attributes: %v OrderingKey: %s DeliveryAttempt: %d calledDone: %v doneFunc: %T size: %d", m.Id, m.Data, m.Attributes, m.OrderingKey, m.DeliveryAttempt, m.calledDone, m.DoneFunc, m.Size)
}

// note that id should be uuid.
func NewMessage(data []byte) *Message {
	return NewEventwithOrderKey(data, "")
}

func NewEventwithOrderKey(data []byte, key string) *Message {
	return &Message{
		Id:          uidGen.Next(),
		Data:        data,
		Attributes:  make(utils.Header),
		OrderingKey: key,
	}

}

// Ack indicates successful processing of a Message passed to the Subscriber.Receive callback.
// It should not be called on any other Message value.
// If message acknowledgement fails, the Message will be redelivered.
// Client code must call Ack or Nack when finished for each received Message.
// Calls to Ack or Nack have no effect after the first call.
func (m *Message) Ack() {
	m.done(true)
}

// Nack indicates that the client will not or cannot process a Message passed to the Subscriber.Receive callback.
// It should not be called on any other Message value.
// Nack will result in the Message being redelivered more quickly than if it were allowed to expire.
// Client code must call Ack or Nack when finished for each received Message.
// Calls to Ack or Nack have no effect after the first call.
func (m *Message) Nack() {
	m.done(false)
}

func (m *Message) done(ack bool) {
	if m.calledDone {
		return
	}
	m.calledDone = true
	m.DoneFunc(m.Id, ack)
}

// todo: consider compile them with protobuf
// hint: just codec the Event struct
func ToByte(m *Message) []byte {
	mb, _ := Encode(m)
	return mb
}

func ToMessage(bytes []byte) (*Message, error) {
	m := &Message{}
	err := Decode(bytes, m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func Encode(data interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Decode(data []byte, to interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(to)
}
