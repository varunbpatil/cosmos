package cosmos

import (
	"context"
)

type CommandService interface {
	Spec(ctx context.Context, connector *Connector) (*Message, error)
	Check(ctx context.Context, connector *Connector, config interface{}) (*Message, error)
	Discover(ctx context.Context, connector *Connector, config interface{}) (*Message, error)
	Read(ctx context.Context, connector *Connector, empty bool) (<-chan interface{}, <-chan error)
	Write(ctx context.Context, connector *Connector, in <-chan *Message) (<-chan interface{}, <-chan error)
	Normalize(ctx context.Context, connector *Connector, basicNormalization bool) (<-chan interface{}, <-chan error)
}
