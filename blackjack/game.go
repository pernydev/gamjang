package blackjack

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Game struct {
	Deck       *Deck
	PlayerID   string
	Bet        int
	PlayerHand *Hand
	DealerHand *Hand

	PlayerStand bool
	PlayerBust  bool
}

func (g *Game) Render() string {
	playerHand := ""
	for _, card := range g.PlayerHand.Cards {
		playerHand += fmt.Sprintf("%s ", card.String())
	}
	playerHand += fmt.Sprintf("(%d)", g.PlayerHand.Total())
	dealerHand := ""
	for _, card := range g.DealerHand.Cards {
		dealerHand += fmt.Sprintf("%s ", card.String())
	}
	dealerHand += fmt.Sprintf("(%d)", g.DealerHand.Total())
	output := fmt.Sprintf("\n\n**Your hand:** %s\n**Dealer's hand:** %s\n\n**Bet:** %d <:coin:1356375500632756224>\n", playerHand, dealerHand, g.Bet)
	if g.PlayerBust {
		output += "**You busted!**\n"
	}
	return output
}

func (g *Game) Hit() bool {
	fmt.Println("Hit")
	ok, c := g.Deck.Draw()
	if !ok {
		return false
	}
	g.PlayerHand.AddCard(c)
	if g.PlayerHand.Total() == 21 {
		// we do not allow the player to get blackjack, since that means they win
		g.PlayerHand.Cards = g.PlayerHand.Cards[:len(g.PlayerHand.Cards)-1]
		return g.Hit()
	}
	if g.PlayerHand.Total() > 21 {
		g.PlayerBust = true
		g.PlayerStand = true
		deleteGame(g.PlayerID)
	}
	return true
}

func (g *Game) Stand() bool {
	fmt.Println("Stand")
	g.PlayerStand = true
	for g.DealerHand.Total() < 17 {
		ok, c := g.Deck.Draw()
		if !ok {
			return false
		}
		g.DealerHand.AddCard(c)
		if g.DealerHand.Total() > 16 && (g.DealerHand.Total() > 21 || g.DealerHand.Total()-1 < g.PlayerHand.Total()) {
			g.DealerHand.Cards = g.DealerHand.Cards[:len(g.DealerHand.Cards)-1]
			return g.Stand()
		}
	}
	deleteGame(g.PlayerID)
	return true
}

func (g *Game) RenderButtons() []discordgo.MessageComponent {
	if g.PlayerStand {
		return []discordgo.MessageComponent{}
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Style:    discordgo.PrimaryButton,
					Label:    "Hit",
					CustomID: "hit",
				},
				discordgo.Button{
					Style:    discordgo.PrimaryButton,
					Label:    "Stand",
					CustomID: "stand",
				},
			},
		},
	}
}

func NewGame(playerID string, bet int) *Game {
	deck := NewDeck()
	deck.Shuffle()

	playerHand := &Hand{}
	dealerHand := &Hand{}
	ok, c := deck.Draw()
	if !ok {
		return nil
	}
	playerHand.AddCard(c)
	ok, c = deck.Draw()
	if !ok {
		return nil
	}
	playerHand.AddCard(c)
	ok, c = deck.Draw()
	if !ok {
		return nil
	}
	dealerHand.AddCard(c)
	ok, c = deck.Draw()
	if !ok {
		return nil
	}
	if playerHand.Total() == 21 {
		return NewGame(playerID, bet)
	}
	g := &Game{
		Deck:        deck,
		PlayerID:    playerID,
		Bet:         bet,
		PlayerHand:  playerHand,
		DealerHand:  dealerHand,
		PlayerStand: false,
		PlayerBust:  false,
	}
	saveGame(g)
	return g
}
