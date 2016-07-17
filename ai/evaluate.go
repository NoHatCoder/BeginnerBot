package ai

import (
	"fmt"
	"io"
	"text/tabwriter"

	"../bitboard"
	"../tak"
)

const (
	endgameCutoff = 7
)

type FlatScores struct {
	Hard, Soft int
}

type Weights struct {
	TopFlat     int
	EndgameFlat int
	Standing    int
	Capstone    int

	FlatCaptives     FlatScores
	StandingCaptives FlatScores
	CapstoneCaptives FlatScores

	Liberties int

	Groups [8]int
}

var defaultWeights = Weights{
	TopFlat:     400,
	EndgameFlat: 800,
	Standing:    200,
	Capstone:    300,

	FlatCaptives: FlatScores{
		Hard: 125,
		Soft: -75,
	},
	StandingCaptives: FlatScores{
		Hard: 125,
		Soft: -50,
	},
	CapstoneCaptives: FlatScores{
		Hard: 150,
		Soft: -25,
	},

	Liberties: 20,

	Groups: [8]int{
		0,   // 0
		0,   // 1
		0,   // 2
		100, // 3
		300, // 4
	},
}

var defaultWeights6 = Weights{
	TopFlat:     400,
	EndgameFlat: 800,
	Standing:    200,
	Capstone:    300,

	FlatCaptives: FlatScores{
		Hard: 125,
		Soft: -75,
	},
	StandingCaptives: FlatScores{
		Hard: 125,
		Soft: -50,
	},
	CapstoneCaptives: FlatScores{
		Hard: 150,
		Soft: -25,
	},

	Liberties: 20,

	Groups: [8]int{
		0,   // 0
		0,   // 1
		0,   // 2
		100, // 3
		300, // 4
		500, // 5
	},
}

var DefaultWeights = []Weights{
	defaultWeights,  // 0
	defaultWeights,  // 1
	defaultWeights,  // 2
	defaultWeights,  // 3
	defaultWeights,  // 4
	defaultWeights,  // 5
	defaultWeights6, // 6
	defaultWeights,  // 7
	defaultWeights,  // 8
}

func MakeEvaluator(size int, w *Weights) EvaluationFunc {
	if w == nil {
		w = &DefaultWeights[size]
	}
	if true {
		return func(m *MinimaxAI, p *tak.Position) int64 {
			return evaluateNohat(WeightNohat, m, p, size)
		}
	} else {
		return func(m *MinimaxAI, p *tak.Position) int64 {
			return evaluate(w, m, p)
		}
	}
}

func evaluateTerminal(p *tak.Position, winner tak.Color) int64 {
	var pieces int64
	if winner == tak.White {
		pieces = int64(p.WhiteStones())
	} else {
		pieces = int64(p.BlackStones())
	}
	switch winner {
	case tak.NoColor:
		return 0
	case p.ToMove():
		return MaxEval - int64(p.MoveNumber()) + pieces
	default:
		return MinEval + int64(p.MoveNumber()) - pieces
	}
}

func EvaluateWinner(m *MinimaxAI, p *tak.Position) int64 {
	if over, winner := p.GameOver(); over {
		return evaluateTerminal(p, winner)
	}
	return 0
}

var WeightNohat = []float64{
	0, //0
	-20,
	-40,
	-80,
	-160,
	-320,
	-640,
	-100000, //Opponent road
	0,       //8
	25,
	50,
	100,
	200,
	400,
	10000,
	20000, //Own road
	600,   //16 //Flat
	450,   //Captured, same as top
	300,   //Captured, different
	150,   //Capstone reserve
	0,     //20 //Stack potential 0
	20,    //Stack potential 1
	100,   //Stack potential 2
	200,   //Stack additional potential
	750,   //24 //Early wall bonus
	-10,   //offedge penalty
	-30}   //edge penalty

func evaluateNohat(w []float64, m *MinimaxAI, p *tak.Position, size int) int64 {
	size2 := size * size
	sizefactor := 1 / float64(size)
	var path uint64
	var roads [4][36]uint8
	var seeker uint64 = 5<<49 + 1<<uint(50-size) + 1<<uint(50+size)
	var spreadmask uint64 = (1 << uint(size)) - 1
	for a := 0; a < 4; a++ {
		for b := 0; b < size2; b++ {
			roads[a][b] = 7
		}
	}
	var directions = [4]int{1, -size, -1, size}
	var leftmask uint64 = 0
	var rightmask uint64 = 0
	var topmask uint64 = 0
	var bottommask uint64 = 0
	for a := 0; a < size; a++ {
		leftmask += 1 << uint(a*size)
		rightmask += 1 << uint(a*size+size-1)
		topmask += 1 << uint(size2-size+a)
		bottommask += 1 << uint(a)
	}
	var edge1mask = leftmask + rightmask
	var edge2mask = topmask + bottommask
	var offedge1mask = (leftmask << 1) + (rightmask >> 1)
	var offedge2mask = (topmask >> uint(size)) + (bottommask << uint(size))
	var path1 func(spot int, direction int, whitein uint8, blackin uint8) (uint8, uint8)
	path1 = func(spot int, direction int, whitein uint8, blackin uint8) (uint8, uint8) {
		var spotmask uint64 = 1 << uint(spot)
		var newdirection int
		var newspot int
		var newspotmask uint64
		var whiteout uint8 = 7
		var blackout uint8 = 7
		var newwhiteout uint8
		var newblackout uint8
		if p.White&spotmask != 0 {
			if p.Standing&spotmask != 0 {
				blackin += 4
				whitein += 2
			} else if p.Caps&spotmask != 0 {
				blackin += 4
			} else {
				blackin += 2
			}
		} else if p.Black&spotmask != 0 {
			if p.Standing&spotmask != 0 {
				whitein += 4
				blackin += 2
			} else if p.Caps&spotmask != 0 {
				whitein += 4
			} else {
				whitein += 2
			}
		} else {
			whitein++
			blackin++
		}
		if whitein > 6 && blackin > 6 {
			return 7, 7
		} else if spotmask&topmask == 0 {
			path ^= spotmask
			for a := 0; a < 4; a++ {
				newdirection = directions[a]
				newspot = spot + newdirection
				if direction+newdirection != 0 && (seeker>>uint(50-newspot)&path&^spotmask) == 0 {
					newspotmask = 1 << uint(newspot)
					if newspot >= size && ((path|newspotmask)&leftmask == 0 || (path|newspotmask)&rightmask == 0) {
						newwhiteout, newblackout = path1(newspot, newdirection, whitein, blackin)
						if newwhiteout < whiteout {
							whiteout = newwhiteout
						}
						if newblackout < blackout {
							blackout = newblackout
						}
					}
				}
			}
			path ^= spotmask
		} else {
			whiteout = whitein
			blackout = blackin
		}
		if roads[0][spot] > whiteout {
			roads[0][spot] = whiteout
		}
		if roads[1][spot] > blackout {
			roads[1][spot] = blackout
		}
		return whiteout, blackout
	}
	var path2 func(spot int, direction int, whitein uint8, blackin uint8) (uint8, uint8)
	path2 = func(spot int, direction int, whitein uint8, blackin uint8) (uint8, uint8) {
		var spotmask uint64 = 1 << uint(spot)
		var newdirection int
		var newspot int
		var newspotmask uint64
		var whiteout uint8 = 7
		var blackout uint8 = 7
		var newwhiteout uint8
		var newblackout uint8
		if p.White&spotmask != 0 {
			if p.Standing&spotmask != 0 {
				blackin += 4
				whitein += 2
			} else if p.Caps&spotmask != 0 {
				blackin += 4
			} else {
				blackin += 2
			}
		} else if p.Black&spotmask != 0 {
			if p.Standing&spotmask != 0 {
				whitein += 4
				blackin += 2
			} else if p.Caps&spotmask != 0 {
				whitein += 4
			} else {
				whitein += 2
			}
		} else {
			whitein++
			blackin++
		}
		if whitein > 6 && blackin > 6 {
			return 7, 7
		} else if spotmask&rightmask == 0 {
			path ^= spotmask
			for a := 0; a < 4; a++ {
				newdirection = directions[a]
				newspot = spot + newdirection
				if newspot >= 0 && newspot < size2 {
					newspotmask = 1 << uint(newspot)
					if direction+newdirection != 0 && ((seeker>>uint(50-newspot)&path&^spotmask) == 0 || newspotmask&rightmask != 0) {
						if newspotmask&leftmask == 0 && ((path|newspotmask)&topmask == 0 || (path|newspotmask)&bottommask == 0) {
							newwhiteout, newblackout = path2(newspot, newdirection, whitein, blackin)
							if newwhiteout < whiteout {
								whiteout = newwhiteout
							}
							if newblackout < blackout {
								blackout = newblackout
							}
						}
					}
				}
			}
			path ^= spotmask
		} else {
			whiteout = whitein
			blackout = blackin
			/*
				for a := 0;a<size;a++{
					for b := 0;b<size;b++{
						fmt.Printf("%d", ((path|spotmask)>>uint(b+a*size))&1)
					}
					fmt.Printf("\n")
				}
				fmt.Printf("\n")
			*/
		}
		if roads[2][spot] > whiteout {
			roads[2][spot] = whiteout
		}
		if roads[3][spot] > blackout {
			roads[3][spot] = blackout
		}
		return whiteout, blackout
	}
	for a := 0; a < size; a++ {
		path = 0
		path1(a, size, 0, 0)
	}
	for a := 0; a < size2; a += size {
		path = 0
		path2(a, 1, 0, 0)
	}
	/*
		fmt.Printf("Boards:\n\n")
		fmt.Printf("%+v\n\n", p)
		for c := 0;c<4;c++{
			for a := 0;a<size;a++{
				for b := 0;b<size;b++{
					fmt.Printf("%d", roads[c][b+a*size])
				}
				fmt.Printf("\n")
			}
			fmt.Printf("\n")
		}
	*/
	/*
		for a := 0;a<size;a++{
			for b := 0;b<size;b++{
				fmt.Printf("%d", (rightmask>>uint(b+a*size))&1)
			}
			fmt.Printf("\n")
		}
	*/
	var offset1 uint8
	var offset2 uint8
	var value float64 = 0
	left := p.WhiteStones()
	if p.BlackStones() < left {
		left = p.BlackStones()
	}
	flatcount := float64(bitboard.Popcount(p.White&^p.Standing)-bitboard.Popcount(p.Black&^p.Standing)) * w[16]
	flatcount += float64(p.WhiteCaps()-p.BlackCaps()) * w[19] * float64(left) / float64(size2)
	flatcount += float64(bitboard.Popcount(p.White&p.Standing)-bitboard.Popcount(p.Black&p.Standing)) * w[24] * float64(left) / float64(size2)
	for a := 0; a < size2; a++ {
		h := p.Height[a]
		if h > 1 {
			topwhite := p.White&(1<<uint(a)) != 0
			blacks := float64(bitboard.Popcount(p.Stacks[a]))
			whites := float64(h) - 1 - blacks
			if topwhite {
				flatcount += whites * w[17]
				flatcount -= blacks * w[18]
			} else {
				flatcount += whites * w[18]
				flatcount -= blacks * w[17]
			}
		}
	}

	flatcount += float64(bitboard.Popcount(p.White&offedge1mask)+bitboard.Popcount(p.White&offedge2mask)-bitboard.Popcount(p.Black&offedge1mask)-bitboard.Popcount(p.Black&offedge2mask)) * w[25]
	flatcount += float64(bitboard.Popcount(p.White&edge1mask)+bitboard.Popcount(p.White&edge2mask)-bitboard.Popcount(p.Black&edge1mask)-bitboard.Popcount(p.Black&edge2mask)) * w[26]

	for a := 0; a < size; a++ {
		for b := 0; b < size; b++ {
			spot := b + size*a
			h := int(p.Height[spot])
			if h >= 1 {
				longoption := true
				if h > size {
					h = size + 1
					longoption = false
				}
				topwhite := p.White&(1<<uint(spot)) != 0
				topcap := p.Caps&(1<<uint(spot)) != 0 && ((p.Stacks[spot]&1 == 1) != topwhite || h == 1)
				blacks := bitboard.Popcount(p.Stacks[spot] & spreadmask)
				whites := h - 1 - blacks
				var spread int
				if topwhite {
					spread = whites
				} else {
					spread = blacks
				}
				if longoption {
					spread++
				}
				var potential int = 0
				var potentialvalue float64
				countspread := func(direction int, endmask uint64) {
					var newpotential int = 0
					searchspot := spot
					for c := 0; c < spread && (1<<uint(searchspot))&endmask == 0; c++ {
						searchspot += direction
						var searchspotmask uint64 = 1 << uint(searchspot)
						if p.Caps&searchspotmask != 0 || (p.Standing&searchspotmask != 0 && !topcap) {
							break
						} else if p.White&searchspotmask != 0 {
							if !topwhite {
								newpotential += 2
							}
						} else if p.Black&searchspotmask != 0 {
							if topwhite {
								newpotential += 2
							}
						} else {
							newpotential++
						}
						if longoption && c+1 == spread {
							newpotential--
						}
						if newpotential > potential {
							potential = newpotential
						}
						if p.Standing&searchspotmask != 0 {
							break
						}
					}
				}
				countspread(1, rightmask)
				countspread(size, topmask)
				countspread(-size, bottommask)
				countspread(-1, leftmask)
				if potential == 0 {
					potentialvalue += w[20]
				} else if potential == 1 {
					potentialvalue += w[21]
				} else {
					potentialvalue += w[22] + w[23]*float64(potential-2)
				}
				if topwhite {
					flatcount += potentialvalue
				} else {
					flatcount -= potentialvalue
				}
			}
		}
	}
	if p.ToMove() == tak.White {
		offset1 = 15
		offset2 = 7
		value += flatcount
	} else {
		offset1 = 7
		offset2 = 15
		value -= flatcount
	}
	for c := 0; c < 4; c += 2 {
		for a := 0; a < size; a++ {
			for b := 0; b < size; b++ {
				value += w[offset1-roads[c][b+a*size]] * sizefactor
				value += w[offset2-roads[c+1][b+a*size]] * sizefactor
			}
		}
	}
	/*latefactor := 1+float64(p.MoveNumber()/20)
	if latefactor>2{
		latefactor=2
	}*/
	//value+=m.rand.Float64()*800*latefactor
	value += m.rand.Float64()
	if over, winner := p.GameOver(); over {
		switch winner {
		case tak.NoColor:
			return 0
		case p.ToMove():
			return MaxEval - int64(p.MoveNumber())*1000000 + int64(value)
		default:
			return MinEval + int64(p.MoveNumber())*1000000 + int64(value)
		}
	}
	var searchpos *tak.Position
	var err error
	for _, move := range p.Threatmoves {
		searchpos, err = p.MovePreallocated(&move, searchpos)
		if err == nil {
			over, winner := searchpos.GameOver()
			if over && searchpos.ToMove() != winner {
				return int64(value) + 8000
			}
		}
	}
	//fmt.Printf("%+v\n\n", value)
	return int64(value)
}

func evaluate(w *Weights, m *MinimaxAI, p *tak.Position) int64 {
	if over, winner := p.GameOver(); over {
		return evaluateTerminal(p, winner)
	}

	var ws, bs int64

	analysis := p.Analysis()
	fmt.Printf("%+v\n", p)

	left := p.WhiteStones()
	if p.BlackStones() < left {
		left = p.BlackStones()
	}
	if left > endgameCutoff {
		left = endgameCutoff
	}
	flat := w.TopFlat + ((endgameCutoff-left)*w.EndgameFlat)/endgameCutoff
	if p.ToMove() == tak.White {
		ws += int64(flat/2) + 50
	} else {
		bs += int64(flat/2) + 50
	}

	ws += int64(bitboard.Popcount(p.White&^p.Caps&^p.Standing) * flat)
	bs += int64(bitboard.Popcount(p.Black&^p.Caps&^p.Standing) * flat)
	ws += int64(bitboard.Popcount(p.White&p.Standing) * w.Standing)
	bs += int64(bitboard.Popcount(p.Black&p.Standing) * w.Standing)
	ws += int64(bitboard.Popcount(p.White&p.Caps) * w.Capstone)
	bs += int64(bitboard.Popcount(p.Black&p.Caps) * w.Capstone)

	for i, h := range p.Height {
		if h <= 1 {
			continue
		}
		bit := uint64(1 << uint(i))
		s := p.Stacks[i] & ((1 << (h - 1)) - 1)
		var hf, sf int
		var ptr *int64
		if p.White&bit != 0 {
			sf = bitboard.Popcount(s)
			hf = int(h) - sf - 1
			ptr = &ws
		} else {
			hf = bitboard.Popcount(s)
			sf = int(h) - hf - 1
			ptr = &bs
		}

		switch {
		case p.Standing&(1<<uint(i)) != 0:
			*ptr += (int64(hf*w.StandingCaptives.Hard) +
				int64(sf*w.StandingCaptives.Soft))
		case p.Caps&(1<<uint(i)) != 0:
			*ptr += (int64(hf*w.CapstoneCaptives.Hard) +
				int64(sf*w.CapstoneCaptives.Soft))
		default:
			*ptr += (int64(hf*w.FlatCaptives.Hard) +
				int64(sf*w.FlatCaptives.Soft))
		}
	}

	ws += int64(m.scoreGroups(analysis.WhiteGroups, w))
	bs += int64(m.scoreGroups(analysis.BlackGroups, w))

	wr := p.White &^ p.Standing
	br := p.Black &^ p.Standing
	wl := bitboard.Popcount(bitboard.Grow(&m.c, ^p.Black, wr) &^ p.White)
	bl := bitboard.Popcount(bitboard.Grow(&m.c, ^p.White, br) &^ p.Black)
	ws += int64(w.Liberties * wl)
	bs += int64(w.Liberties * bl)

	if p.ToMove() == tak.White {
		return ws - bs
	}
	return bs - ws
}

func (ai *MinimaxAI) scoreGroups(gs []uint64, ws *Weights) int {
	sc := 0
	for _, g := range gs {
		w, h := bitboard.Dimensions(&ai.c, g)

		sc += ws.Groups[w]
		sc += ws.Groups[h]
	}

	return sc
}

func ExplainScore(m *MinimaxAI, out io.Writer, p *tak.Position) {
	tw := tabwriter.NewWriter(out, 4, 8, 1, '\t', 0)
	fmt.Fprintf(tw, "\twhite\tblack\n")
	var scores [2]struct {
		flats    int
		standing int
		caps     int

		stones   int
		captured int
	}

	scores[0].flats = bitboard.Popcount(p.White &^ p.Caps &^ p.Standing)
	scores[1].flats = bitboard.Popcount(p.Black &^ p.Caps &^ p.Standing)
	scores[0].standing = bitboard.Popcount(p.White & p.Standing)
	scores[1].standing = bitboard.Popcount(p.Black & p.Standing)
	scores[0].caps = bitboard.Popcount(p.White & p.Caps)
	scores[1].caps = bitboard.Popcount(p.Black & p.Caps)

	for i, h := range p.Height {
		if h <= 1 {
			continue
		}
		s := p.Stacks[i] & ((1 << (h - 1)) - 1)
		bf := bitboard.Popcount(s)
		wf := int(h) - bf - 1
		scores[0].stones += wf
		scores[1].stones += bf

		captured := int(h - 1)
		if captured > p.Size()-1 {
			captured = p.Size() - 1
		}
		if p.White&(1<<uint(i)) != 0 {
			scores[0].captured += captured
		} else {
			scores[1].captured += captured
		}
	}

	fmt.Fprintf(tw, "flats\t%d\t%d\n", scores[0].flats, scores[1].flats)
	fmt.Fprintf(tw, "standing\t%d\t%d\n", scores[0].standing, scores[1].standing)
	fmt.Fprintf(tw, "caps\t%d\t%d\n", scores[0].caps, scores[1].caps)
	fmt.Fprintf(tw, "captured\t%d\t%d\n", scores[0].captured, scores[1].captured)
	fmt.Fprintf(tw, "stones\t%d\t%d\n", scores[0].stones, scores[1].stones)

	analysis := p.Analysis()

	wr := p.White &^ p.Standing
	br := p.Black &^ p.Standing
	wl := bitboard.Popcount(bitboard.Grow(&m.c, ^p.Black, wr) &^ p.White)
	bl := bitboard.Popcount(bitboard.Grow(&m.c, ^p.White, br) &^ p.Black)

	fmt.Fprintf(tw, "liberties\t%d\t%d\n", wl, bl)

	for i, g := range analysis.WhiteGroups {
		w, h := bitboard.Dimensions(&m.c, g)
		fmt.Fprintf(tw, "g%d\t%dx%x\n", i, w, h)
	}
	for i, g := range analysis.BlackGroups {
		w, h := bitboard.Dimensions(&m.c, g)
		fmt.Fprintf(tw, "g%d\t\t%dx%x\n", i, w, h)
	}
	tw.Flush()
}
