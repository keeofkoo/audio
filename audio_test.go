package audio_test

import (
	"io"
	"testing"

	"github.com/pipelined/audio"
	"github.com/pipelined/signal"
	"github.com/stretchr/testify/assert"
)

func TestAsset(t *testing.T) {
	tests := []struct {
		asset       *audio.Asset
		numChannels int
		sampleRate  signal.SampleRate
		value       float64
		messages    int
		samples     int
	}{
		{
			asset:       &audio.Asset{},
			numChannels: 1,
			value:       0.5,
			messages:    10,
			samples:     100,
		},
		{
			asset:       &audio.Asset{},
			numChannels: 2,
			value:       0.7,
			messages:    100,
			samples:     1000,
		},
		{
			asset:       &audio.Asset{},
			numChannels: 0,
			value:       0.7,
			messages:    0,
			samples:     0,
		},
	}
	bufferSize := 10

	for _, test := range tests {
		fn, err := test.asset.Sink("", int(test.sampleRate), test.numChannels)
		assert.Nil(t, err)
		assert.NotNil(t, fn)
		for i := 0; i < test.messages; i++ {
			buf := signal.Float64Buffer(test.numChannels, bufferSize, test.value)
			err := fn(buf)
			assert.Nil(t, err)
		}
		assert.Equal(t, test.numChannels, test.asset.NumChannels())
		assert.Equal(t, test.sampleRate, test.asset.SampleRate())
		assert.Equal(t, test.samples, test.asset.Data().Size())
	}
}

func TestTrack(t *testing.T) {
	sampleRate := signal.SampleRate(44100)
	asset1 := audio.SignalAsset(sampleRate, [][]float64{{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}})
	asset2 := audio.SignalAsset(sampleRate, [][]float64{{2, 2, 2, 2, 2, 2, 2, 2, 2, 2}})
	asset3 := &audio.Asset{}
	tests := []struct {
		clips    []audio.Clip
		clipsAt  []int
		expected [][]float64
		msg      string
	}{
		{
			clips: []audio.Clip{
				asset1.Clip(3, 1),
				asset2.Clip(5, 3),
			},
			clipsAt:  []int{3, 4},
			expected: [][]float64{{0, 0, 0, 1, 2, 2, 2, 0}},
			msg:      "Sequence",
		},
		{
			clips: []audio.Clip{
				asset1.Clip(3, 1),
				asset2.Clip(5, 3),
			},
			clipsAt:  []int{3, 4},
			expected: [][]float64{{0, 0, 0, 1, 2, 2, 2, 0}},
			msg:      "Sequence increased bufferSize",
		},
		{
			clips: []audio.Clip{
				asset1.Clip(3, 1),
				asset2.Clip(5, 3),
			},
			clipsAt:  []int{2, 3},
			expected: [][]float64{{0, 0, 1, 2, 2, 2}},
			msg:      "Sequence shifted left",
		},
		{
			clips: []audio.Clip{
				asset1.Clip(3, 1),
				asset2.Clip(5, 3),
			},
			clipsAt:  []int{2, 4},
			expected: [][]float64{{0, 0, 1, 0, 2, 2, 2, 0}},
			msg:      "Sequence with interval",
		},
		{
			clips: []audio.Clip{
				asset1.Clip(3, 3),
				asset2.Clip(5, 2),
			},
			clipsAt:  []int{3, 2},
			expected: [][]float64{{0, 0, 2, 2, 1, 1}},
			msg:      "Overlap previous",
		},
		{
			clips: []audio.Clip{
				asset1.Clip(3, 3),
				asset2.Clip(5, 2),
			},
			clipsAt:  []int{2, 4},
			expected: [][]float64{{0, 0, 1, 1, 2, 2}},
			msg:      "Overlap next",
		},
		{
			clips: []audio.Clip{
				asset1.Clip(3, 5),
				asset2.Clip(5, 2),
			},
			clipsAt:  []int{2, 4},
			expected: [][]float64{{0, 0, 1, 1, 2, 2, 1, 0}},
			msg:      "Overlap single in the middle",
		},
		{
			clips: []audio.Clip{
				asset1.Clip(3, 2),
				asset1.Clip(3, 2),
				asset2.Clip(5, 2),
			},
			clipsAt:  []int{2, 5, 4},
			expected: [][]float64{{0, 0, 1, 1, 2, 2, 1, 0}},
			msg:      "Overlap two in the middle",
		},
		{
			clips: []audio.Clip{
				asset1.Clip(3, 2),
				asset1.Clip(5, 2),
				asset2.Clip(3, 2),
			},
			clipsAt:  []int{2, 5, 3},
			expected: [][]float64{{0, 0, 1, 2, 2, 1, 1, 0}},
			msg:      "Overlap two in the middle shifted",
		},
		{
			clips: []audio.Clip{
				asset1.Clip(3, 2),
				asset2.Clip(3, 5),
			},
			clipsAt:  []int{2, 2},
			expected: [][]float64{{0, 0, 2, 2, 2, 2, 2, 0}},
			msg:      "Overlap single completely",
		},
		{
			clips: []audio.Clip{
				asset1.Clip(3, 2),
				asset1.Clip(5, 2),
				asset2.Clip(1, 8),
			},
			clipsAt:  []int{2, 5, 1},
			expected: [][]float64{{0, 2, 2, 2, 2, 2, 2, 2, 2, 0}},
			msg:      "Overlap two completely",
		},
		{
			expected: [][]float64{},
			msg:      "Empty",
		},
		{
			clips: []audio.Clip{
				asset3.Clip(3, 2),
				asset3.Clip(5, 2),
				asset3.Clip(1, 8),
			},
			clipsAt:  []int{2, 5, 1},
			expected: [][]float64{},
			msg:      "Empty asset clips",
		},
	}
	bufferSize := 2
	for _, test := range tests {
		track := audio.NewTrack(sampleRate, asset1.NumChannels())
		err := track.Reset("")
		assert.Nil(t, err)

		for i, clip := range test.clips {
			track.AddClip(test.clipsAt[i], clip)
		}

		fn, pumpSampleRate, _, err := track.Pump("")
		assert.Equal(t, int(sampleRate), pumpSampleRate)
		assert.Nil(t, err)
		assert.NotNil(t, fn)

		var result, buf signal.Float64
		for err == nil {
			buf, err = fn(bufferSize)
			result = result.Append(buf)
		}
		assert.Equal(t, io.EOF, err)

		assert.Equal(t, len(test.expected), result.NumChannels(), test.msg)
		for i := 0; i < len(result); i++ {
			assert.Equal(t, len(test.expected[i]), len(result[i]), test.msg)
			for j := 0; j < len(result[i]); j++ {
				assert.Equal(t, test.expected[i][j], result[i][j], test.msg)
			}
		}
	}
}
