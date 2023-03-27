package waveform

import (
	"context"
	"os"

	"github.com/go-audio/wav"
	"github.com/stemstr/storage/internal/encoder"
)

type Generator interface {
	Waveform(context.Context, string) ([]int, error)
}

func New(enc encoder.Encoder) Generator {
	return &generator{enc: enc}
}

type generator struct {
	enc encoder.Encoder
}

func (g *generator) Waveform(ctx context.Context, wavFile string) ([]int, error) {
	return getWaveformData(wavFile)
}

func getWaveformData(wavFilepath string) ([]int, error) {
	f, err := os.Open(wavFilepath)
	if err != nil {
		return nil, err
	}

	d := wav.NewDecoder(f)
	b, err := d.FullPCMBuffer()
	if err != nil {
		return nil, err
	}

	chunkSize := len(b.Data) / 64
	pcmMin, pcmMax := 0, 0
	var samples []int
	var bucketData []int
	bucket := chunkSize
	for i, sample := range b.Data {
		if sample < pcmMin {
			pcmMin = sample
		}
		if sample > pcmMax {
			pcmMax = sample
		}

		if i < bucket {
			bucketData = append(bucketData, sample)
		} else {
			// Average the bucketData, add to samples, reset
			n := 0
			for _, v := range bucketData {
				n += v
			}
			avg := int(n / len(bucketData))
			samples = append(samples, avg)
			bucketData = []int{}
			bucket += chunkSize
		}
	}

	min, max := 0, 0
	for _, x := range samples {
		if x < min {
			min = x
		}
		if x > max {
			max = x
		}
	}

	var normalizedSamples []int
	for _, x := range samples {
		// Now normalize the values between 1-100
		//sample := int(1 + ((x-(min))*(100-1))/(max-(min)))
		sample := int(1 + ((x-(min))*(80-16))/(max-(min)))
		normalizedSamples = append(normalizedSamples, sample)
	}

	return normalizedSamples, nil
}
