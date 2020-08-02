package audio_test

import (
	"context"
	"testing"

	"pipelined.dev/audio"
	"pipelined.dev/pipe"
	"pipelined.dev/pipe/mock"
	"pipelined.dev/signal"
)

func TestTrack(t *testing.T) {
	channels := 1
	alloc := signal.Allocator{
		Channels: channels,
		Capacity: 10,
		Length:   10,
	}
	samples1 := alloc.Float64()
	signal.WriteFloat64([]float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}, samples1)
	samples2 := alloc.Float64()
	signal.WriteFloat64([]float64{20, 21, 22, 23, 24, 25, 26, 27, 28, 29}, samples2)

	sampleRate := signal.SampleRate(44100)
	asset1 := audio.SignalAsset(sampleRate, samples1)
	asset2 := audio.SignalAsset(sampleRate, samples2)
	// asset3 := &audio.Asset{}

	type clip struct {
		position int
		audio.Clip
	}
	tests := []struct {
		clips    []clip
		expected []float64
		msg      string
	}{
		{
			clips: []clip{
				{3, asset1.Clip(3, 1)},
				{4, asset2.Clip(5, 3)},
			},
			expected: []float64{0, 0, 0, 13, 25, 26, 27},
			msg:      "Sequence",
		},
		{
			clips: []clip{
				{2, asset1.Clip(3, 1)},
				{3, asset2.Clip(5, 3)},
			},
			expected: []float64{0, 0, 13, 25, 26, 27},
			msg:      "Sequence shifted left",
		},
		{
			clips: []clip{
				{2, asset1.Clip(3, 1)},
				{4, asset2.Clip(5, 3)},
			},
			expected: []float64{0, 0, 13, 0, 25, 26, 27},
			msg:      "Sequence with interval",
		},
		{
			clips: []clip{
				{3, asset1.Clip(3, 3)},
				{2, asset2.Clip(5, 2)},
			},
			expected: []float64{0, 0, 25, 26, 14, 15},
			msg:      "Overlap previous",
		},
		{
			clips: []clip{
				{2, asset1.Clip(3, 3)},
				{4, asset2.Clip(5, 2)},
			},
			expected: []float64{0, 0, 13, 14, 25, 26},
			msg:      "Overlap next",
		},
		{
			clips: []clip{
				{2, asset1.Clip(3, 5)},
				{4, asset2.Clip(5, 2)},
			},
			expected: []float64{0, 0, 13, 14, 25, 26, 17},
			msg:      "Overlap single in the middle",
		},
		{
			clips: []clip{
				{2, asset1.Clip(3, 2)},
				{5, asset1.Clip(3, 2)},
				{4, asset2.Clip(5, 2)},
			},
			expected: []float64{0, 0, 13, 14, 25, 26, 14},
			msg:      "Overlap two in the middle",
		},
		{
			clips: []clip{
				{2, asset1.Clip(3, 2)},
				{5, asset1.Clip(5, 2)},
				{3, asset2.Clip(3, 2)},
			},
			expected: []float64{0, 0, 13, 23, 24, 15, 16},
			msg:      "Overlap two in the middle shifted",
		},
		{
			clips: []clip{
				{2, asset1.Clip(3, 2)},
				{2, asset2.Clip(3, 5)},
			},
			expected: []float64{0, 0, 23, 24, 25, 26, 27},
			msg:      "Overlap single completely",
		},
		{
			clips: []clip{
				{2, asset1.Clip(3, 2)},
				{5, asset1.Clip(5, 2)},
				{1, asset2.Clip(1, 8)},
			},
			expected: []float64{0, 21, 22, 23, 24, 25, 26, 27, 28},
			msg:      "Overlap two completely",
		},
		// {
		// 	expected: []float64{},
		// 	msg:      "Empty",
		// },
		// panics
		// {
		// 	clips: []clip{
		// 		{2, asset3.Clip(3, 2)},
		// 		{5, asset3.Clip(5, 2)},
		// 		{1, asset3.Clip(1, 8)},
		// 	},
		// 	expected: []float64{},
		// 	msg:      "Empty asset clips",
		// },
	}

	for _, test := range tests {
		track := audio.NewTrack(sampleRate, channels)
		for _, clip := range test.clips {
			track.AddClip(clip.position, clip.Clip)
		}

		sink := &mock.Sink{}

		l, _ := pipe.Routing{
			Source: track.Source(0, 0),
			Sink:   sink.Sink(),
		}.Line(2)

		pipe.New(context.Background(), pipe.WithLines(l)).Wait()

		result := make([]float64, sink.Values.Len())
		signal.ReadFloat64(sink.Values, result)

		assertEqual(t, test.msg, result, test.expected)
	}
}
