package parser

import (
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/poolpOrg/earmuff/lexer"
	"github.com/poolpOrg/earmuff/midi"
	"github.com/poolpOrg/earmuff/types"
	"github.com/poolpOrg/go-harmony/chords"
	"github.com/poolpOrg/go-harmony/notes"
)

type Parser struct {
	s   *lexer.Scanner
	buf struct {
		tok lexer.Token
		lit string
		n   int
	}
}

func NewParser(r io.Reader) *Parser {
	return &Parser{s: lexer.NewScanner(r)}
}

// scan returns the next token from the underlying scanner.
// If a token has been unscanned then read that instead.
func (p *Parser) scan() (tok lexer.Token, lit string) {
	// If we have a token on the buffer, then return it.
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}

	// Otherwise read the next token from the scanner.
	tok, lit = p.s.Scan()

	// Save it to the buffer in case we unscan later.
	p.buf.tok, p.buf.lit = tok, lit

	return
}

// unscan pushes the previously read token back onto the buffer.
func (p *Parser) unscan() { p.buf.n = 1 }

// scanIgnoreWhitespace scans the next non-whitespace token.
func (p *Parser) scanIgnoreWhitespace() (tok lexer.Token, lit string) {
	for {
		tok, lit = p.scan()
		if tok == lexer.WHITESPACE || tok == lexer.COMMENT {
			continue
		}
		return tok, lit
	}
}

func (p *Parser) parseBpm() (float64, error) {
	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.NUMBER && tok != lexer.FLOAT {
		return 0, fmt.Errorf("found %q, expected number", lit)
	}

	beats, err := strconv.ParseFloat(p.buf.lit, 64)
	if err != nil {
		return 0, err
	}

	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.SEMICOLON {
		return 0, fmt.Errorf("found %q, expected ;", lit)
	}

	return beats, nil
}

func (p *Parser) parseTimeSignature() (*types.Signature, error) {
	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.NUMBER {
		return nil, fmt.Errorf("found %q, expected number", lit)
	}
	beats, err := strconv.ParseUint(p.buf.lit, 10, 8)
	if err != nil {
		return nil, err
	}

	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.NUMBER {
		return nil, fmt.Errorf("found %q, expected number", lit)
	}
	duration, err := strconv.ParseUint(p.buf.lit, 10, 8)
	if err != nil {
		return nil, err
	}

	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.SEMICOLON {
		return nil, fmt.Errorf("found %q, expected ;", lit)
	}
	return types.NewSignature(uint8(beats), uint8(duration)), nil
}

func (p *Parser) parseProject() (*types.Project, error) {
	project := types.NewProject()

	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.PROJECT {
		return nil, fmt.Errorf("found %q, expected PROJECT", lit)
	}

	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.IDENTIFIER && tok != lexer.STRING {
		return nil, fmt.Errorf("found %q, expected project name", lit)
	} else {
		project.SetName(lit)
	}

	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.BRACKET_OPEN {
		return nil, fmt.Errorf("found %q, expected {", lit)
	}

	for {
		tok, lit := p.scanIgnoreWhitespace()
		if tok == lexer.BRACKET_CLOSE {
			break
		}
		switch tok {
		case lexer.BPM:
			bpm, err := p.parseBpm()
			if err != nil {
				return nil, err
			}
			project.SetBPM(bpm)

		case lexer.TIME:
			timeSignature, err := p.parseTimeSignature()
			if err != nil {
				return nil, err
			}
			project.SetSignature(timeSignature)

		case lexer.TRACK:
			track, err := p.parseTrack(project)
			if err != nil {
				return nil, err
			}
			project.AddTrack(track)

		case lexer.COPYRIGHT:
			text, err := p.parseCopyright()
			if err != nil {
				return nil, err
			}
			project.AddCopyright(text)

		case lexer.TEXT:
			text, err := p.parseText()
			if err != nil {
				return nil, err
			}
			project.AddText(text)

		default:
			return nil, fmt.Errorf("found %q, expected TRACK or }", lit)
		}
	}

	//	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.BRACKET_CLOSE {
	//		return nil, fmt.Errorf("found %q, expected }", lit)
	//	}

	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.EOF {
		return nil, fmt.Errorf("found %q, expected EOF", lit)
	}

	return project, nil
}

func (p *Parser) parseTrack(project *types.Project) (*types.Track, error) {
	track := types.NewTrack()
	track.SetBPM(project.GetBPM())
	track.SetSignature(project.GetSignature())

	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.IDENTIFIER && tok != lexer.STRING {
		return nil, fmt.Errorf("found %q, expected track name", lit)
	} else {
		track.SetName(lit)
	}

	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.BRACKET_OPEN {
		return nil, fmt.Errorf("found %q, expected {", lit)
	}

	for {
		tok, lit := p.scanIgnoreWhitespace()
		if tok == lexer.BRACKET_CLOSE {
			break
		}
		switch tok {
		case lexer.BPM:
			_, err := p.parseBpm()
			if err != nil {
				return nil, err
			}
		case lexer.TIME:
			_, err := p.parseTimeSignature()
			if err != nil {
				return nil, err
			}
		case lexer.BAR:
			bar, err := p.parseBar(track)
			if err != nil {
				return nil, err
			}
			track.AddBar(bar)

		case lexer.COPYRIGHT:
			text, err := p.parseCopyright()
			if err != nil {
				return nil, err
			}
			track.AddCopyright(text)

		case lexer.INSTRUMENT:
			text, err := p.parseInstrument()
			if err != nil {
				return nil, err
			}
			track.SetInstrument(text)

		case lexer.TEXT:
			text, err := p.parseText()
			if err != nil {
				return nil, err
			}
			track.AddText(text)
		default:
			return nil, fmt.Errorf("found %q, expected BAR or }", lit)
		}
	}
	return track, nil
}

/*
	func (p *Parser) parseTrack(project *types.Project) (*types.Track, error) {
		track := types.NewTrack()
		track.SetBPM(project.GetBPM())
		track.SetSignature(project.GetSignature())

		if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.BRACKET_OPEN {
			return nil, fmt.Errorf("found %q, expected {", lit)
		}

		for {
			tok, lit := p.scanIgnoreWhitespace()
			if tok == lexer.BRACKET_CLOSE {
				break
			}
			switch tok {
			case lexer.BPM:
				_, err := p.parseBpm()
				if err != nil {
					return nil, err
				}
			case lexer.TIME:
				_, err := p.parseTimeSignature()
				if err != nil {
					return nil, err
				}
			case lexer.BAR:
				bar, err := p.parseBar(track)
				if err != nil {
					return nil, err
				}
				track.AddBar(bar)
			default:
				return nil, fmt.Errorf("found %q, expected BAR or }", lit)
			}
		}
		return track, nil
	}
*/
func (p *Parser) parseBar(track *types.Track) (*types.Bar, error) {
	bar := types.NewBar(uint32(len(track.GetBars())))
	bar.SetBPM(track.GetBPM())
	bar.SetSignature(track.GetSignature())

	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.BRACKET_OPEN {
		return nil, fmt.Errorf("found %q, expected {", lit)
	}

	for {
		tok, lit := p.scanIgnoreWhitespace()
		if tok == lexer.BRACKET_CLOSE {
			break
		}
		switch tok {
		case lexer.BPM:
			_, err := p.parseBpm()
			if err != nil {
				return nil, err
			}
		case lexer.TIME:
			_, err := p.parseTimeSignature()
			if err != nil {
				return nil, err
			}

		case lexer.TEXT:
			text, err := p.parseText()
			if err != nil {
				return nil, err
			}
			bar.AddText(text)

		case lexer.ON:
			err := p.parseOn(bar)
			if err != nil {
				return nil, err
			}

		default:
			return nil, fmt.Errorf("found %q, expected TIME, BEAT or }", lit)
		}
	}

	return bar, nil
}

func (p *Parser) parsePlayable(bar *types.Bar, duration uint16, beat uint8, delta float64) (types.Playable, error) {
	var playable types.Playable
	for {
		tok, lit := p.scanIgnoreWhitespace()
		if tok == lexer.SEMICOLON {
			break
		}
		switch tok {
		case lexer.CHORD:
			chord, err := p.parseChord()
			if err != nil {
				return nil, err
			}
			chord.SetDuration(duration)
			playable = chord
		case lexer.NOTE:
			note, err := p.parseNote()
			if err != nil {
				return nil, err
			}
			note.SetDuration(duration)
			playable = note

		case lexer.CYMBAL:
			//n, _ := notes.Parse("B2")	// acoustic bass drum (35)
			//n, _ := notes.Parse("A6") // open triangle (81)
			n, _ := notes.Parse("D#4") // Ride Cymbal 1 (51)

			note := types.NewNote(*n)
			note.SetDuration(duration)
			playable = note

		case lexer.SNARE:
			//n, _ := notes.Parse("B2")	// acoustic bass drum (35)
			//n, _ := notes.Parse("A6") // open triangle (81)
			n, _ := notes.Parse("D3") // acoustic snare (38)

			note := types.NewNote(*n)
			note.SetDuration(duration)
			playable = note

		case lexer.OPEN_HI_HAT:
			//n, _ := notes.Parse("B2")	// acoustic bass drum (35)
			//n, _ := notes.Parse("A6") // open triangle (81)
			n, _ := notes.Parse("A#3") // acoustic snare (38)

			note := types.NewNote(*n)
			note.SetDuration(duration)
			playable = note

		default:
			return nil, fmt.Errorf("found %q, expected ;", lit)
		}

		ticksPerBeat := uint32(960)
		ticksPerBar := uint32(bar.GetSignature().GetBeats()) * ticksPerBeat
		ticksPerSubdivision := uint32(ticksPerBeat) / uint32(bar.GetSignature().GetDuration())

		deltaTicks := float64(ticksPerSubdivision) * delta
		tick := (bar.GetOffset() * ticksPerBar) +
			uint32(beat-1)*uint32(ticksPerBeat) +
			uint32(deltaTicks)

		playable.SetTick(tick)

		for {
			tok, _ := p.scanIgnoreWhitespace()
			if tok == lexer.SEMICOLON {
				p.unscan()
				break
			}
			switch tok {
			case lexer.VELOCITY:
				if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.NUMBER {
					return nil, fmt.Errorf("found %q, expected NUMBER", lit)
				} else {
					tmp, err := strconv.ParseUint(lit, 10, 8)
					if err != nil {
						return nil, err
					}
					playable.SetVelocity(uint8(tmp))
				}
			}
		}

	}
	return playable, nil
}

func (p *Parser) parseChord() (*types.Chord, error) {
	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.IDENTIFIER {
		return nil, fmt.Errorf("found %q, expected chord name", lit)
	} else {
		chord, err := chords.Parse(lit)
		if err != nil {
			return nil, err
		}
		return types.NewChord(*chord), nil
	}
}

func (p *Parser) parseNote() (*types.Note, error) {
	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.IDENTIFIER {
		return nil, fmt.Errorf("found %q, expected note name", lit)
	} else {
		note, err := notes.Parse(lit)
		if err != nil {
			return nil, err
		}
		return types.NewNote(*note), nil
	}
}

func (p *Parser) parseCopyright() (string, error) {
	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.IDENTIFIER && tok != lexer.STRING {
		return "", fmt.Errorf("found %q, expected string", lit)
	} else {
		if tok, _ := p.scanIgnoreWhitespace(); tok != lexer.SEMICOLON {
			return "", fmt.Errorf("found %q, expected ;", lit)
		}
		return lit, nil
	}
}

func (p *Parser) parseText() (string, error) {
	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.IDENTIFIER && tok != lexer.STRING {
		return "", fmt.Errorf("found %q, expected string", lit)
	} else {
		if tok, _ := p.scanIgnoreWhitespace(); tok != lexer.SEMICOLON {
			return "", fmt.Errorf("found %q, expected ;", lit)
		}
		return lit, nil
	}
}

func (p *Parser) parseName() (string, error) {
	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.IDENTIFIER && tok != lexer.STRING {
		return "", fmt.Errorf("found %q, expected string", lit)
	} else {
		if tok, _ := p.scanIgnoreWhitespace(); tok != lexer.SEMICOLON {
			return "", fmt.Errorf("found %q, expected ;", lit)
		}
		return lit, nil
	}
}

func (p *Parser) parseInstrument() (string, error) {
	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.IDENTIFIER && tok != lexer.STRING {
		return "", fmt.Errorf("found %q, expected string", lit)
	} else {
		if tok, _ := p.scanIgnoreWhitespace(); tok != lexer.SEMICOLON {
			return "", fmt.Errorf("found %q, expected ;", lit)
		}

		_, err := midi.InstrumentToPC(lit)
		if err != nil {
			return "", fmt.Errorf("found %q, unknown instrument", lit)
		}

		return lit, nil
	}
}

func (p *Parser) parseOn(bar *types.Bar) error {
	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.BEAT {
		return fmt.Errorf("found %q, expected beat", lit)
	}

	var beat uint8
	var delta float64
	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.NUMBER && tok != lexer.FLOAT {
		return fmt.Errorf("found %q, expected NUMBER", lit)
	} else if tok == lexer.NUMBER {
		tmp, err := strconv.ParseUint(lit, 10, 8)
		if err != nil {
			return err
		}
		if tmp == 0 || uint64(tmp) > uint64(bar.GetSignature().GetBeats()) {
			return fmt.Errorf("no such beat: %d", tmp)
		}
		beat = uint8(tmp)
		delta = 0.0
	} else {
		tmp, err := strconv.ParseFloat(lit, 32)
		if err != nil {
			return err
		}

		integer, fraction := math.Modf(tmp)

		if uint64(integer) == 0 || uint64(integer) > uint64(bar.GetSignature().GetBeats()) {
			return fmt.Errorf("no such beat: %d", tmp)
		}
		beat = uint8(integer)
		delta = fraction
	}

	if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.PLAY {
		return fmt.Errorf("found %q, expected PLAY", lit)
	}
	return p.parsePlay(bar, beat, delta)
}

func (p *Parser) parsePlay(bar *types.Bar, beat uint8, delta float64) error {

	tok, lit := p.scanIgnoreWhitespace()

	switch tok {
	case lexer.WHOLE:
		playable, err := p.parsePlayable(bar, 1, beat, delta)
		if err != nil {
			return err
		}
		bar.AddPlayable(playable)
	case lexer.HALF:
		playable, err := p.parsePlayable(bar, 2, beat, delta)
		if err != nil {
			return err
		}
		bar.AddPlayable(playable)
	case lexer.QUARTER:
		playable, err := p.parsePlayable(bar, 4, beat, delta)
		if err != nil {
			return err
		}
		bar.AddPlayable(playable)
	case lexer.NUMBER:
		value, err := strconv.ParseUint(lit, 10, 16)
		if err != nil {
			return err
		}

		if value == 8 {
			if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.TH {
				return fmt.Errorf("found %q, expected note name", lit)
			}
		} else if value == 16 {
			if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.TH {
				return fmt.Errorf("found %q, expected note name", lit)
			}
		} else if value == 32 {
			if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.ND {
				return fmt.Errorf("found %q, expected note name", lit)
			}
		} else if value == 64 {
			if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.TH {
				return fmt.Errorf("found %q, expected note name", lit)
			}
		} else if value == 128 {
			if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.TH {
				return fmt.Errorf("found %q, expected note name", lit)
			}
		} else if value == 256 {
			if tok, lit := p.scanIgnoreWhitespace(); tok != lexer.TH {
				return fmt.Errorf("found %q, expected note name", lit)
			}
		} else {
			return fmt.Errorf("found %q, expected value", lit)
		}
		playable, err := p.parsePlayable(bar, uint16(value), beat, delta)
		if err != nil {
			return err
		}
		bar.AddPlayable(playable)
	default:
		return fmt.Errorf("found %q, expected TIME, BEAT or }", lit)
	}

	return nil
}

func (p *Parser) Parse() (*types.Project, error) {
	return p.parseProject()
}
