package output

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gordonklaus/portaudio"
)

const sampleRate uint = 44100
const numSamples uint = 2000

// AudioOutput is a output stream to audio
type AudioOutput struct {
	*portaudio.Stream
	strong, weak           chan struct{}
	strongSound, weakSound []float64
	outputDeviceName       string
}

// NewAudioOutput returns a new AudioOutput instance with default values
func NewAudioOutput(strongFreq, weakFreq float64) *AudioOutput {
	return &AudioOutput{
		Stream:      nil,
		strong:      make(chan struct{}, 1),
		weak:        make(chan struct{}, 1),
		strongSound: GenerateSin(sampleRate, numSamples, strongFreq),
		weakSound:   GenerateSin(sampleRate, numSamples, weakFreq),
	}
}

// NewAudioOutputWithDevice returns a new AudioOutput instance and selects the output device by name or index.
func NewAudioOutputWithDevice(strongFreq, weakFreq float64, outputDeviceName string) *AudioOutput {
	o := NewAudioOutput(strongFreq, weakFreq)
	o.outputDeviceName = outputDeviceName
	return o
}

// Start starts the output channel
func (o *AudioOutput) Start() (err error) {
	if err = portaudio.Initialize(); err != nil {
		return
	}

	outDevice, err := resolveOutputDevice(o.outputDeviceName)
	if err != nil {
		return err
	}

	params := portaudio.HighLatencyParameters(nil, outDevice)
	params.Output.Channels = 1
	params.SampleRate = float64(sampleRate)
	params.FramesPerBuffer = 0

	o.Stream, err = portaudio.OpenStream(params, o.processAudio)
	if err != nil {
		return
	}

	return o.Stream.Start()
}

// Stop stops the audio output
func (o *AudioOutput) Stop() error {
	// make sure to terminate the audio device and delete the stream!
	defer portaudio.Terminate()
	defer func() {
		o.Stream = nil
	}()

	err := o.Stream.Stop()
	if err != nil {
		return err
	}

	return o.Stream.Close()
}

func (o *AudioOutput) processAudio(b []float32) {
	data := make([]float64, len(b))

	select {
	case <-o.strong:
		data = o.strongSound[:len(b)]
	case <-o.weak:
		data = o.weakSound[:len(b)]
	default:
	}

	for i := range b {
		b[i] = float32(data[i] * 2)
	}
}

// PlayStrong plays a accent note for full bars
func (o *AudioOutput) PlayStrong() {
	if o.Stream == nil {
		panic(errors.New("AudioOutput is not started yet or terminated"))
	}

	o.strong <- struct{}{}
}

// PlayWeak plays a mediate sound sample for 4ths etc.
func (o *AudioOutput) PlayWeak() {
	if o.Stream == nil {
		panic(errors.New("AudioOutput is not started yet or terminated"))
	}

	o.weak <- struct{}{}
}

func resolveOutputDevice(nameOrIndex string) (*portaudio.DeviceInfo, error) {
	if strings.TrimSpace(nameOrIndex) == "" {
		return portaudio.DefaultOutputDevice()
	}

	devices, err := portaudio.Devices()
	if err != nil {
		return nil, err
	}

	if idx, err := strconv.Atoi(nameOrIndex); err == nil {
		if idx < 0 || idx >= len(devices) {
			return nil, fmt.Errorf("output device index %d out of range", idx)
		}
		if devices[idx].MaxOutputChannels == 0 {
			return nil, fmt.Errorf("device %d has no output channels", idx)
		}
		return devices[idx], nil
	}

	lower := strings.ToLower(nameOrIndex)
	for _, dev := range devices {
		if dev.MaxOutputChannels == 0 {
			continue
		}
		if strings.Contains(strings.ToLower(dev.Name), lower) {
			return dev, nil
		}
	}
	return nil, fmt.Errorf("no output device matching %q", nameOrIndex)
}
