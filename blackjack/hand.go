package blackjack

type Hand struct {
	Cards []Card
}

func (h *Hand) Total() int {
	total := 0
	aces := 0

	for _, c := range h.Cards {
		total += c.BlackjackValue()
		if c.Value == "A" {
			aces++
		}
	}

	// Adjust for aces if total exceeds 21
	for aces > 0 && total > 21 {
		total -= 10
		aces--
	}

	return total
}

func (h *Hand) AddCard(c Card) {
	h.Cards = append(h.Cards, c)
}
