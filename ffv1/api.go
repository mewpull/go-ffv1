package ffv1

import (
	"fmt"
	"sync"
)

type Decoder struct {
	width            uint32
	height           uint32
	record           configRecord
	state_transition [256]uint8
	initial_states   [][][]uint8
	current_frame    internalFrame
}

type Frame struct {
	Buf              [][]byte
	Buf16            [][]uint16
	Width            uint32
	Height           uint32
	BitDepth         uint8
	ColorSpace       int
	HasChroma        bool
	HasAlpha         bool
	ChromaSubsampleV uint8
	ChromaSubsampleH uint8
}

func NewDecoder(record []byte, width uint32, height uint32) (*Decoder, error) {
	ret := new(Decoder)

	if width == 0 || height == 0 {
		return nil, fmt.Errorf("invalid dimensions: %dx%d", width, height)
	}

	ret.width = width
	ret.height = height

	err := parseConfigRecord(record, &ret.record)
	if err != nil {
		return nil, fmt.Errorf("invalid v3 configuration record: %s", err.Error())
	}

	ret.initializeStates()

	return ret, nil
}

func (d *Decoder) DecodeFrame(frame []byte) (*Frame, error) {

	// Allocate and fill frame info
	ret := new(Frame)
	ret.Width = d.width
	ret.Height = d.height
	ret.BitDepth = d.record.bits_per_raw_sample
	ret.ColorSpace = int(d.record.colorspace_type)
	ret.HasChroma = d.record.chroma_planes
	ret.HasAlpha = d.record.extra_plane
	if ret.HasChroma {
		ret.ChromaSubsampleV = d.record.log2_v_chroma_subsample
		ret.ChromaSubsampleH = d.record.log2_h_chroma_subsample
	}

	numPlanes := 1
	if d.record.chroma_planes {
		numPlanes += 2
	}
	if d.record.extra_plane {
		numPlanes++
	}

	// Hideous and temporary.
	if d.record.bits_per_raw_sample == 8 {
		ret.Buf = make([][]byte, numPlanes)
		ret.Buf[0] = make([]byte, int(d.width*d.height))
		if d.record.chroma_planes {
			chromaWidth := d.width >> d.record.log2_h_chroma_subsample
			chromaHeight := d.height >> d.record.log2_v_chroma_subsample
			ret.Buf[1] = make([]byte, int(chromaWidth*chromaHeight))
			ret.Buf[2] = make([]byte, int(chromaWidth*chromaHeight))
		}
		if d.record.extra_plane {
			ret.Buf[3] = make([]byte, int(d.width*d.height))
		}
	} else {
		ret.Buf16 = make([][]uint16, numPlanes)
		ret.Buf16[0] = make([]uint16, int(d.width*d.height))
		if d.record.chroma_planes {
			chromaWidth := d.width >> d.record.log2_h_chroma_subsample
			chromaHeight := d.height >> d.record.log2_v_chroma_subsample
			ret.Buf16[1] = make([]uint16, int(chromaWidth*chromaHeight))
			ret.Buf16[2] = make([]uint16, int(chromaWidth*chromaHeight))
		}
		if d.record.extra_plane {
			ret.Buf16[3] = make([]uint16, int(d.width*d.height))
		}
	}

	d.current_frame.keyframe = isKeyframe(frame)

	err := d.parseFooters(frame, &d.current_frame)
	if err != nil {
		return nil, fmt.Errorf("invalid frame footer: %s", err.Error())
	}

	// Slice threading lazymode
	errs := make([]error, len(d.current_frame.slices))
	wg := new(sync.WaitGroup)
	for i := 0; i < len(d.current_frame.slices); i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, errs []error, n int) {
			errs[n] = d.decodeSlice(frame, &d.current_frame, n, ret)
			wg.Done()
		}(wg, errs, i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("slice %s failed: %s", i, err.Error())
		}
	}

	return ret, nil
}
