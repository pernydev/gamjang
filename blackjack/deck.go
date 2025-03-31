package blackjack

import (
	"fmt"
	"strconv"
)

type Card struct {
	Suit  string
	Value string
}

func (c *Card) BlackjackValue() int {
	switch c.Value {
	case "A":
		return 11
	case "J", "Q", "K":
		return 10
	default:
		val, err := strconv.Atoi(c.Value)
		if err != nil {
			fmt.Printf("Invalid card value: %s\n", c.Value)
			return 0
		}
		return val
	}
}
