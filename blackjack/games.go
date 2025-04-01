package blackjack

import "fmt"

var games = make(map[string]*Game)

func saveGame(g *Game) {
	fmt.Println("saveGame", g.PlayerID)
	fmt.Println("saveGame", games)
	games[g.PlayerID] = g
}
func GetGame(playerID string) *Game {
	fmt.Println("GetGame", playerID)
	fmt.Println("GetGame", games)
	return games[playerID]
}
func deleteGame(playerID string) {
	delete(games, playerID)
}
