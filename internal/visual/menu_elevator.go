package visual

// Four seconds before launch, and five after a safe return. The sprite is
// clipped by the elevator well while below deck; it never slides down the
// outside of the carrier. Closed doors expose the original carrier artwork.
const sortieLiftTime = 4.0

func (r *Raster) idleElevator(age float64, lowering bool) {
	if age < 0 || age > 5 {
		return
	}
	opening := smooth(age / .8)
	lift := smooth((age - .8) / 2.5)
	if lowering {
		lift = 1 - smooth((age-.8)/2.5)
		opening *= 1 - smooth((age-3.3)/.9)
	} else {
		opening *= 1 - smooth((age-3.3)/.7)
	}
	left, right := 30-6*opening, 30+6*opening
	if opening > .01 {
		r.idleRect(left, 28.5, right-left, 2.5, 0x071017)
		r.idleLine(left, 28.5, left, 31, 0x506876)
		r.idleLine(right, 28.5, right, 31, 0x304753)
		r.idleLine(left, 31, right, 31, 0x152b38)
		platform := 29 + 7*(1-lift)
		if platform < 31 {
			r.idleLine(left, platform, right, platform, 0x849d9f)
		}
		r.idleLine(left-1, 29, left-1, 29, 0xbaa16d)
	}
	jet := New(r.W, r.H/2)
	jet.idleJet(30, 27+7*(1-lift), -1, true, false)
	scale := float64(r.W) / 80
	for i, c := range jet.Pixels {
		if c == 0 {
			continue
		}
		x, y := float64(i%r.W)/scale, float64(i/r.W)/scale
		if y <= 29 || (opening > .01 && x >= left && x <= right && y <= 31) {
			r.Pixels[i] = c
		}
	}
}
