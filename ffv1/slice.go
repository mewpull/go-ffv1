package ffv1

import (
	"fmt"
	"math"

	"github.com/dwbuiten/go-ffv1/ffv1/rangecoder"
)

type internalFrame struct {
	keyframe   bool
	slice_info []sliceInfo
	slices     []slice
}

type sliceInfo struct {
	pos  int
	size uint32
}

type slice struct {
	header  sliceHeader
	start_x uint32
	start_y uint32
	width   uint32
	height  uint32
}

type sliceHeader struct {
	slice_width_minus1    uint32
	slice_height_minus1   uint32
	slice_x               uint32
	slice_y               uint32
	quant_table_set_index []uint8
	picture_structure     uint8
	sar_num               uint32
	sar_den               uint32
}

func countSlices(buf []byte, header *internalFrame, ec bool) error {
	footerSize := 3
	if ec {
		footerSize += 5
	}

	endPos := len(buf)
	for endPos > 0 {
		var info sliceInfo

		size := uint32(buf[endPos-footerSize]) << 16
		size |= uint32(buf[endPos-footerSize+1]) << 8
		size |= uint32(buf[endPos-footerSize+2])

		info.size = size
		info.pos = endPos - int(size) - footerSize
		header.slice_info = append([]sliceInfo{info}, header.slice_info...) //prepend
		endPos = info.pos
	}

	if endPos < 0 {
		return fmt.Errorf("invalid slice footer")
	}

	return nil
}

func (d *Decoder) parseFooters(buf []byte, header *internalFrame) error {
	err := countSlices(buf, header, d.record.ec != 0)
	if err != nil {
		return fmt.Errorf("couldn't count slices: %s", err.Error())
	}

	header.slices = make([]slice, len(header.slice_info))

	return nil
}

func (d *Decoder) parseSliceHeader(c *rangecoder.Coder, s *slice) {
	slice_state := make([]uint8, ContextSize)
	for i := 0; i < ContextSize; i++ {
		slice_state[i] = 128
	}

	s.header.slice_x = c.UR(slice_state)
	s.header.slice_y = c.UR(slice_state)
	s.header.slice_width_minus1 = c.UR(slice_state)
	s.header.slice_height_minus1 = c.UR(slice_state)

	quant_table_set_index_count := 1
	if d.record.chroma_planes {
		quant_table_set_index_count++
	}
	if d.record.extra_plane {
		quant_table_set_index_count++
	}

	s.header.quant_table_set_index = make([]uint8, quant_table_set_index_count)

	for i := 0; i < quant_table_set_index_count; i++ {
		s.header.quant_table_set_index[i] = uint8(c.UR(slice_state))
	}

	s.header.picture_structure = uint8(c.UR(slice_state))
	s.header.sar_num = c.UR(slice_state)
	s.header.sar_den = c.UR(slice_state)

	// Calculate bounaries for easy use elsewhere
	s.start_x = s.header.slice_x * d.width / (uint32(d.record.num_h_slices_minus1) + 1)
	s.start_y = s.header.slice_y * d.height / (uint32(d.record.num_v_slices_minus1) + 1)
	s.width = ((s.header.slice_x + s.header.slice_width_minus1 + 1) * d.width / (uint32(d.record.num_h_slices_minus1) + 1)) - s.start_x
	s.height = ((s.header.slice_y + s.header.slice_height_minus1 + 1) * d.height / (uint32(d.record.num_v_slices_minus1) + 1)) - s.start_y
}

func (d *Decoder) decodeSliceContent(c *rangecoder.Coder, si *sliceInfo, s *slice, frame *Frame) {
	if d.record.colorspace_type != 0 {
		panic("only YCbCr support")
	}

	primary_color_count := 1
	chroma_planes := 0
	if d.record.chroma_planes {
		chroma_planes = 2
		primary_color_count += 2
	}
	if d.record.extra_plan {
		primary_color_count++
	}

	for p := 0; p < primary_color_count; p++ {
		var plane_pixel_height int
		var plane_pixel_width int
		if p == 0 || p == 1+chroma_planes {
			plane_pixel_height = s.height
			plane_pixel_width = s.width
		} else {
			// This is, of course, silly, but I want to do it "by the spec".
			plane_pixel_height = int(math.Ceil(float64(s.height) / float64(1<<d.record.log2_v_chroma_subsample)))
			plane_pixel_width = int(math.Ceil(float64(s.width) / float64(1<<d.record.log2_h_chroma_subsample)))
		}

		for y := 0; i < plane_pixel_height; y++ {
			// Line()
			for x := 0; x < plane_pixel_width; x++ {
				//continue here
			}
		}
	}
}

func (d *Decoder) decodeSlice(buf []byte, header *internalFrame, slicenum int, frame *Frame) error {
	c := rangecoder.NewCoder(buf[header.slice_info[slicenum].pos:])

	state := make([]uint8, ContextSize)
	for i := 0; i < ContextSize; i++ {
		state[i] = 128
	}

	if slicenum == 0 {
		header.keyframe = c.BR(state)
		fmt.Println("keyframe = ", header.keyframe)
	}

	if d.record.coder_type == 2 { // Custom state transition table
		c.SetTable(d.state_transition)
	}

	d.parseSliceHeader(c, &header.slices[slicenum])
	fmt.Println(header.slices[slicenum])

	if d.record.coder_type != 1 && d.record.coder_type != 2 {
		panic("golomb not implemented yet")
	}

	//TODO: Coder types!
	d.decodeSliceContent(c, &header.slice_info[slicenum], &header.slices[slicenum], frame)

	return nil
}
