package components

import (
	"math"
	"strconv"
)

func ratingStars(rating string) int {
	value, err := strconv.ParseFloat(rating, 64)
	if err != nil || value <= 0 {
		return 0
	}
	count := int(math.Round(value))
	if count > 5 {
		return 5
	}
	return count
}
