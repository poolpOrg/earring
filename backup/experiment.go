package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

const (
	SampleRate = 44100
)

type Tuning struct {
	name string
}

type Note struct {
	tuning *Tuning
	name   string
	octave uint8
}

const (
	Whole                = 1
	Half                 = 2
	Quarter              = 4
	Eighth               = 8
	Sixteenth            = 16
	ThirtySecond         = 32
	SixtyFourth          = 64
	HundredTwentyEighth  = 128
	TwoHundredFiftySixth = 256
)

func NewTuning(name string) (*Tuning, error) {
	if name != "a440" {
		return nil, fmt.Errorf("only a440 supported for now")
	}
	return &Tuning{name: name}, nil
}

func (tuning *Tuning) Offset(name string) (int8, error) {
	switch name {
	case "C":
		return 0, nil

	case "C#":
		fallthrough
	case "Db":
		return 1, nil

	case "D":
		return 2, nil

	case "D#":
		fallthrough
	case "Eb":
		return 3, nil

	case "E":
		return 4, nil

	case "F":
		return 5, nil

	case "F#":
		fallthrough
	case "Gb":
		return 6, nil

	case "G":
		return 7, nil

	case "G#":
		fallthrough
	case "Ab":
		return 8, nil

	case "A":
		return 9, nil

	case "A#":
		fallthrough
	case "Bb":
		return 10, nil

	case "B":
		return 11, nil

	}
	return 0, fmt.Errorf("no such note")
}

var a440 = [][]float64{
	{16.35, 17.32, 18.35, 19.45, 20.60, 21.83, 23.12, 24.50, 25.96, 27.50, 29.14, 30.87},
	{32.70, 34.65, 36.71, 38.89, 41.20, 43.65, 46.25, 49.00, 51.91, 55.00, 58.27, 61.74},
	{65.41, 69.30, 73.42, 77.78, 82.41, 87.31, 92.50, 98.00, 103.83, 110.00, 116.54, 123.47},
	{130.81, 138.59, 146.83, 155.56, 164.81, 174.61, 185.00, 196.00, 207.65, 220.00, 233.08, 246.94},
	{261.63, 277.18, 293.66, 311.13, 329.63, 349.23, 369.99, 392.00, 415.30, 440.00, 466.16, 493.88},
	{523.25, 554.37, 587.33, 622.25, 659.25, 698.46, 739.99, 783.99, 830.61, 880.00, 932.33, 987.77},
	{1046.50, 1108.73, 1174.66, 1244.51, 1318.51, 1396.91, 1479.98, 1567.98, 1661.22, 1760.00, 1864.66, 1973.53},
	{2093.00, 2217.46, 2349.32, 2489.02, 2637.02, 2793.83, 2959.96, 3135.96, 3322.44, 3520.00, 3729.31, 3951.07},
	{4186.01, 4434.92, 4698.63, 4978.03, 5274.04, 5587.65, 5919.91, 6271.93, 6644.88, 7040.00, 7458.62, 7902.13},
}

func (tuning *Tuning) Frequency(note string, octave uint8) (float64, error) {
	offset, err := tuning.Offset(note)
	if err != nil {
		return 0.0, err
	}
	if octave > 8 {
		return 0.0, fmt.Errorf("can't produce octave > 8")
	}
	return a440[octave][offset], nil
}

func (tuning *Tuning) Note(name string, octave uint8) *Note {
	return &Note{tuning: tuning, name: name, octave: octave}
}

func (note *Note) Frequency() float64 {
	freq, _ := note.tuning.Frequency(note.name, note.octave)
	return freq
}

func (note *Note) Play(duration time.Duration) {
	sr := beep.SampleRate(SampleRate)
	done := make(chan bool)
	speaker.Play(beep.Seq(beep.Take(sr.N(duration), note), beep.Callback(func() {
		done <- true
	})))
	<-done
}

func (note Note) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		sample := math.Sin((math.Pi * 2 / float64(SampleRate)) * note.Frequency() * float64(i))
		samples[i][0] = sample
		samples[i][1] = sample
	}
	return len(samples), true
}
func (note Note) Err() error {
	return nil
}

type TimeSignature struct {
	unit  uint8
	beats uint8
}

func NewTimeSignature(unit uint8, beats uint8) *TimeSignature {
	return &TimeSignature{unit: unit, beats: beats}
}

type Measure struct {
	timeSignature *TimeSignature
	notes         []Note
}

func NewMeasure(timeSignature *TimeSignature) *Measure {
	return &Measure{timeSignature: timeSignature, notes: make([]Note, 0)}
}

func (measure *Measure) Add(note Note, duration uint8) {
	measure.notes = append(measure.notes, note)
}

func (measure *Measure) Play() {
	for _, note := range measure.notes {
		note.Play(time.Second)
	}
}

func main() {
	sr := beep.SampleRate(SampleRate)
	speaker.Init(SampleRate, sr.N(time.Second/10))

	tuning, err := NewTuning("a440")
	if err != nil {
		log.Fatal(err)
	}

	timeSignature := NewTimeSignature(4, 4)
	measure := NewMeasure(timeSignature)
	measure.Add(*tuning.Note("A", 4), Sixteenth)
	measure.Add(*tuning.Note("B", 4), Quarter)
	measure.Add(*tuning.Note("C", 2), Quarter)
	measure.Play()

	//	note.Play(time.Second / 4)
	//
	//	note = tuning.Note("B", 4)
	//	note.Play(time.Second / 4)
	//
	//	note = tuning.Note("A", 4)
	//	note.Play(time.Second / 4)
	//
	//	note = tuning.Note("C", 4)
	//	note.Play(time.Second / 4)
}
