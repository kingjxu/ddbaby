package gto

import "strings"

// Card encoding: (rank-2)*4 + suit, rank 2..14, suit 0..3 (S,H,C,D).
const (
	suitSpade   = 0
	suitHeart   = 1
	suitClub    = 2
	suitDiamond = 3
)

func rankOf(card int) int { return card/4 + 2 }
func suitOf(card int) int { return card % 4 }

// parseCard accepts the ddbaby card format, e.g. "A-H", "10-S", "J-D".
func parseCard(s string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(s), "-")
	if len(parts) != 2 {
		return 0, false
	}
	rank := rankValue(parts[0])
	if rank == 0 {
		return 0, false
	}
	var suit int
	switch strings.ToUpper(strings.TrimSpace(parts[1])) {
	case "S":
		suit = suitSpade
	case "H":
		suit = suitHeart
	case "C":
		suit = suitClub
	case "D":
		suit = suitDiamond
	default:
		return 0, false
	}
	return (rank-2)*4 + suit, true
}

func rankValue(s string) int {
	v := strings.ToUpper(strings.TrimSpace(s))
	switch v {
	case "2", "3", "4", "5", "6", "7", "8", "9":
		return int(v[0] - '0')
	case "10", "T":
		return 10
	case "J":
		return 11
	case "Q":
		return 12
	case "K":
		return 13
	case "A":
		return 14
	}
	return 0
}

func rankChar(rank int) string {
	switch rank {
	case 10:
		return "T"
	case 11:
		return "J"
	case 12:
		return "Q"
	case 13:
		return "K"
	case 14:
		return "A"
	default:
		return string(rune('0' + rank))
	}
}

// handClass returns the 169-grid class: "AA", "AKs", "AKo", "72o", ...
func handClass(a, b string) string {
	ca, okA := parseCard(a)
	cb, okB := parseCard(b)
	if !okA || !okB {
		return ""
	}
	ra, rb := rankOf(ca), rankOf(cb)
	sa, sb := suitOf(ca), suitOf(cb)
	if ra < rb {
		ra, rb = rb, ra
		sa, sb = sb, sa
	}
	if ra == rb {
		return rankChar(ra) + rankChar(rb)
	}
	suffix := "o"
	if sa == sb {
		suffix = "s"
	}
	return rankChar(ra) + rankChar(rb) + suffix
}

type handGroup struct {
	count int
	rank  int
}

// eval5 returns a comparable score for exactly five cards. Higher is better.
// The implementation is allocation-free because it runs inside Monte Carlo.
func eval5(c0, c1, c2, c3, c4 int) int32 {
	cards := [5]int{c0, c1, c2, c3, c4}
	var rs [5]int
	flush := true
	firstSuit := suitOf(cards[0])
	for i, c := range cards {
		rs[i] = rankOf(c)
		if suitOf(c) != firstSuit {
			flush = false
		}
	}
	// Insertion sort ranks descending (five elements).
	for i := 1; i < 5; i++ {
		for j := i; j > 0 && rs[j] > rs[j-1]; j-- {
			rs[j], rs[j-1] = rs[j-1], rs[j]
		}
	}

	var present [15]bool
	for _, r := range rs {
		present[r] = true
	}
	straightHigh := 0
	for high := 14; high >= 5; high-- {
		if present[high] && present[high-1] && present[high-2] && present[high-3] && present[high-4] {
			straightHigh = high
			break
		}
	}
	if straightHigh == 0 && present[14] && present[5] && present[4] && present[3] && present[2] {
		straightHigh = 5 // wheel
	}

	var counts [13]int
	for _, r := range rs {
		counts[r-2]++
	}
	var groups [5]handGroup
	gn := 0
	for rank := 14; rank >= 2; rank-- {
		if counts[rank-2] > 0 {
			groups[gn] = handGroup{counts[rank-2], rank}
			gn++
		}
	}
	// Sort groups by count desc, then rank desc (at most five elements).
	for i := 1; i < gn; i++ {
		for j := i; j > 0; j-- {
			if groups[j].count > groups[j-1].count ||
				(groups[j].count == groups[j-1].count && groups[j].rank > groups[j-1].rank) {
				groups[j], groups[j-1] = groups[j-1], groups[j]
				continue
			}
			break
		}
	}

	score := func(cat int32, kickers ...int32) int32 {
		s := cat << 20
		shift := uint(16)
		for _, k := range kickers {
			s |= k << shift
			if shift < 4 {
				break
			}
			shift -= 4
		}
		return s
	}

	switch {
	case flush && straightHigh > 0:
		return score(8, int32(straightHigh))
	case gn >= 1 && groups[0].count == 4:
		kicker := 0
		for i := 0; i < gn; i++ {
			if groups[i].count != 4 {
				kicker = groups[i].rank
				break
			}
		}
		return score(7, int32(groups[0].rank), int32(kicker))
	case gn >= 2 && groups[0].count == 3 && groups[1].count >= 2:
		return score(6, int32(groups[0].rank), int32(groups[1].rank))
	case flush:
		return score(5, int32(rs[0]), int32(rs[1]), int32(rs[2]), int32(rs[3]), int32(rs[4]))
	case straightHigh > 0:
		return score(4, int32(straightHigh))
	case gn >= 1 && groups[0].count == 3:
		k1, k2 := 0, 0
		for i := 0; i < gn; i++ {
			if groups[i].count != 3 {
				if k1 == 0 {
					k1 = groups[i].rank
				} else if k2 == 0 {
					k2 = groups[i].rank
				}
			}
		}
		return score(3, int32(groups[0].rank), int32(k1), int32(k2))
	case gn >= 2 && groups[0].count == 2 && groups[1].count == 2:
		kicker := 0
		for i := 0; i < gn; i++ {
			if groups[i].count == 1 {
				kicker = groups[i].rank
				break
			}
		}
		return score(2, int32(groups[0].rank), int32(groups[1].rank), int32(kicker))
	case gn >= 1 && groups[0].count == 2:
		var k [3]int32
		kn := 0
		for i := 0; i < gn; i++ {
			if groups[i].count == 1 {
				k[kn] = int32(groups[i].rank)
				kn++
				if kn == 3 {
					break
				}
			}
		}
		return score(1, int32(groups[0].rank), k[0], k[1], k[2])
	default:
		return score(0, int32(rs[0]), int32(rs[1]), int32(rs[2]), int32(rs[3]), int32(rs[4]))
	}
}

// eval7 returns the best five-card score out of seven cards.
func eval7(cards []int) int32 {
	if len(cards) < 7 {
		return 0
	}
	var best int32
	for a := 0; a <= 2; a++ {
		for b := a + 1; b <= 3; b++ {
			for c := b + 1; c <= 4; c++ {
				for d := c + 1; d <= 5; d++ {
					for e := d + 1; e <= 6; e++ {
						s := eval5(cards[a], cards[b], cards[c], cards[d], cards[e])
						if s > best {
							best = s
						}
					}
				}
			}
		}
	}
	return best
}
