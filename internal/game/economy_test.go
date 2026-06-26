package game

import "testing"

func TestSellPrice(t *testing.T) {
	cases := map[int]int{
		0:   1, // worthless still floors at 1
		1:   1,
		2:   1, // 2/2 = 1
		3:   1,
		10:  5,
		75:  37,
		100: 50,
	}
	for value, want := range cases {
		if got := sellPrice(value); got != want {
			t.Errorf("sellPrice(%d) = %d, want %d", value, got, want)
		}
	}
}
