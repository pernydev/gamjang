package blackjack

import (
	"fmt"
	"math/rand"
	"strconv"
)

type Card struct {
	Suit  DeckSuit
	Value DeckValue
}

func (c *Card) BlackjackValue() int {
	switch c.Value {
	case "A":
		return 11
	case "J", "Q", "K":
		return 10
	default:
		val, err := strconv.Atoi(string(c.Value))
		if err != nil {
			fmt.Printf("Invalid card value: %s\n", c.Value)
			return 0
		}
		return val
	}
}

func (c *Card) String() string {
	suit := ""
	switch c.Suit {
	case Hearts:
		suit = "♥"
	case Diamonds:
		suit = "♦"
	case Clubs:
		suit = "♣"
	case Spades:
		suit = "♠"
	default:
		suit = "?"
	}
	return fmt.Sprintf("`%s%s`", suit, c.Value)
}

type DeckSuit string

var (
	Hearts   DeckSuit = "Hearts"
	Diamonds DeckSuit = "Diamonds"
	Clubs    DeckSuit = "Clubs"
	Spades   DeckSuit = "Spades"
)

type DeckValue string

var (
	Ace   DeckValue = "A"
	Two   DeckValue = "2"
	Three DeckValue = "3"
	Four  DeckValue = "4"
	Five  DeckValue = "5"
	Six   DeckValue = "6"
	Seven DeckValue = "7"
	Eight DeckValue = "8"
	Nine  DeckValue = "9"
	Ten   DeckValue = "10"
	Jack  DeckValue = "J"
	Queen DeckValue = "Q"
	King  DeckValue = "K"
)

var (
	DeckValues = []DeckValue{
		Ace,
		Two,
		Three,
		Four,
		Five,
		Six,
		Seven,
		Eight,
		Nine,
		Ten,
		Jack,
		Queen,
		King,
	}
	DeckSuits = []DeckSuit{
		Hearts,
		Diamonds,
		Clubs,
		Spades,
	}
)

type Deck struct {
	Cards []Card
}

func NewDeck() *Deck {
	cards := make([]Card, 0, len(DeckValues)*len(DeckSuits))
	for _, suit := range DeckSuits {
		for _, value := range DeckValues {
			cards = append(cards, Card{Suit: suit, Value: value})
		}
	}
	return &Deck{Cards: cards}
}
func (d *Deck) Shuffle() {
	for i := len(d.Cards) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	}
}
func (d *Deck) Draw() (bool, Card) {
	if len(d.Cards) == 0 {
		return false, Card{}
	}
	card := d.Cards[len(d.Cards)-1]
	d.Cards = d.Cards[:len(d.Cards)-1]
	return true, card
}
