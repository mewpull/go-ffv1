package ffv1

// This file is automatically generated from pred.go using go generate
// Please DO NOT manually modify this file

func deriveBorders16(plane []uint16, x int, y int, width int, height int, stride int) (int, int, int, int, int, int) {
	var T int
	var L int
	var t int
	var l int
	var tr int
	var tl int

	pos := y*stride + x

	// This is really slow and stupid but matches the spec exactly. Each of the
	// neighbouring values has been left entirely separate, and none skipped,
	// even if they could be.
	//
	// Please never implement an actual decoder this way.

	// T
	if y == 0 || y == 1 {
		T = 0
	} else {
		T = int(plane[pos-(2*stride)])
	}

	// L
	if y == 0 {
		if x == 0 || x == 1 {
			L = 0
		} else {
			L = int(plane[pos-2])
		}
	} else {
		if x == 0 {
			L = 0
		} else if x == 1 {
			L = int(plane[pos-(1*stride)-1])
		} else {
			L = int(plane[pos-2])
		}
	}

	// t
	if y == 0 {
		t = 0
	} else {
		t = int(plane[pos-(1*stride)])
	}

	// l
	if y == 0 {
		if x == 0 {
			l = 0
		} else {
			l = int(plane[pos-1])
		}
	} else {
		if x == 0 {
			l = int(plane[pos-(1*stride)])
		} else {
			l = int(plane[pos-1])
		}
	}

	// tl
	if y == 0 {
		tl = 0
	} else {
		if x == 0 {
			if y == 1 {
				tl = 0
			} else {
				tl = int(plane[pos-(2*stride)])
			}
		} else {
			tl = int(plane[pos-(1*stride)-1])
		}
	}

	// tr
	if y == 0 {
		tr = 0
	} else {
		if x == width-1 {
			tr = int(plane[pos-(1*stride)])
		} else {
			tr = int(plane[pos-(1*stride)+1])
		}
	}

	return T, L, t, l, tr, tl
}
