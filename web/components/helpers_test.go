package components

import "testing"

func TestRatingStarsRoundsWithinFiveStarRange(t *testing.T) {
	tests := []struct {
		rating string
		want   int
	}{
		{rating: "4.8", want: 5},
		{rating: "4.3", want: 4},
		{rating: "0", want: 0},
		{rating: "not-a-rating", want: 0},
		{rating: "6", want: 5},
	}
	for _, test := range tests {
		if got := ratingStars(test.rating); got != test.want {
			t.Errorf("ratingStars(%q) = %d, want %d", test.rating, got, test.want)
		}
	}
}
