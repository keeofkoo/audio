package audio

import (
	"context"
	"io"
	"sync/atomic"

	"pipelined.dev/pipe"
	"pipelined.dev/pipe/mutability"
	"pipelined.dev/signal"
)

// Repeater sinks the signal and sources it to multiple pipelines.
type Repeater struct {
	mutability.Mutability
	bufferSize int
	sampleRate signal.Frequency
	channels   int
	sources    []chan message
}

type message struct {
	buffer  signal.Floating
	sources int32
}

// Sink must be called once per repeater.
func (r *Repeater) Sink() pipe.SinkAllocatorFunc {
	return func(ctx context.Context, bufferSize int, props pipe.SignalProperties) (pipe.Sink, error) {
		r.sampleRate = props.SampleRate
		r.channels = props.Channels
		r.bufferSize = bufferSize
		p := signal.GetPoolAllocator(props.Channels, bufferSize, bufferSize)
		return pipe.Sink{
			Mutability: r.Mutability,
			SinkFunc: func(in signal.Floating) error {
				for _, source := range r.sources {
					out := p.GetFloat64()
					signal.FloatingAsFloating(in, out)
					source <- message{
						sources: int32(len(r.sources)),
						buffer:  out,
					}
				}
				return nil
			},
			FlushFunc: func(context.Context) error {
				for i := range r.sources {
					close(r.sources[i])
				}
				r.sources = nil
				return nil
			},
		}, nil
	}
}

// AddOutput adds the line to the repeater. Will panic if repeater is immutable.
func (r *Repeater) AddOutput(runner *pipe.Runner, l *pipe.Line) mutability.Mutation {
	return r.Mutability.Mutate(func() error {
		l.Source = r.Source()
		runner.Push(runner.AddLine(l))
		return nil
	})
}

// Source must be called at least once per repeater.
func (r *Repeater) Source() pipe.SourceAllocatorFunc {
	source := make(chan message, 1)
	r.sources = append(r.sources, source)
	return func(ctx context.Context, bufferSize int) (pipe.Source, pipe.SignalProperties, error) {
		p := signal.GetPoolAllocator(r.channels, bufferSize, bufferSize)
		var (
			message message
			ok      bool
		)
		return pipe.Source{
				SourceFunc: func(b signal.Floating) (int, error) {
					message, ok = <-source
					if !ok {
						return 0, io.EOF
					}
					read := signal.FloatingAsFloating(message.buffer, b)
					if atomic.AddInt32(&message.sources, -1) == 0 {
						message.buffer.Free(p)
					}
					return read, nil
				},
			},
			pipe.SignalProperties{
				SampleRate: r.sampleRate,
				Channels:   r.channels,
			},
			nil
	}
}
