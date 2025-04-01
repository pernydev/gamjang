package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/pernydev/gamjang/blackjack"
	"github.com/redis/go-redis/v9"
)

// Bot parameters
var (
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
)

var s *discordgo.Session
var db *redis.Client
var ctx = context.Background()
var footer = "-# This is an April Fools joke, all currency and games are fake. Do not gamble."

func init() { flag.Parse() }

func init() {
	godotenv.Load()
	var err error
	s, err = discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
	opts, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		panic(err)
	}
	db = redis.NewClient(opts)
}

func renderAmount(amount int) string {
	return fmt.Sprintf("**%d <:coin:1356375500632756224>**", amount)
}

func ptr(s string) *string {
	return &s
}

var (
	minBet = float64(1)
	// Initialize random seed once at startup
	randSource = rand.NewSource(time.Now().UnixNano())
	randGen    = rand.New(randSource)
	commands   = []*discordgo.ApplicationCommand{
		{
			Name:        "help",
			Description: "Show help information about all commands",
		},
		{
			Name: "balance",
			// All commands and options must have a description
			// Commands/options without description will fail the registration
			// of the command.
			Description: "Check your balance",
		},
		{
			Name:        "fountain",
			Description: "Claim 50 coins from the fountain (1 hour cooldown)",
		},
		{
			Name:        "blackjack",
			Description: "Play blackjack",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "bet",
					Description: "Bet amount",
					Type:        discordgo.ApplicationCommandOptionInteger,
					Required:    true,
					MinValue:    &minBet,
				},
			},
		},
		{
			Name:        "roulette",
			Description: "Play roulette",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "bet",
					Description: "Bet amount",
					Type:        discordgo.ApplicationCommandOptionInteger,
					Required:    true,
					MinValue:    &minBet,
				},
				{
					Name:        "type",
					Description: "Type of bet",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "Single Number",
							Value: "number",
						},
						{
							Name:  "Color",
							Value: "color",
						},
						{
							Name:  "Even/Odd",
							Value: "parity",
						},
						{
							Name:  "High/Low",
							Value: "range",
						},
						{
							Name:  "Dozen",
							Value: "dozen",
						},
						{
							Name:  "Column",
							Value: "column",
						},
					},
				},
				{
					Name:        "value",
					Description: "Value to bet on",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"help": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			helpText := "**Available Commands:**\n\n"
			helpText += "`/balance` - Check your current balance\n"
			helpText += "`/fountain` - Claim 50 coins (1 hour cooldown)\n\n"
			helpText += "**Games:**\n"
			helpText += "`/blackjack <bet>` - Play blackjack\n"
			helpText += "`/roulette <bet> <type> <value>` - Play roulette\n\n"
			helpText += "**Roulette Bet Types:**\n"
			helpText += "- `number`: Bet on a single number (0-36), pays 35:1\n"
			helpText += "- `color`: Bet on red/black/green, pays 2:1\n"
			helpText += "- `parity`: Bet on even/odd, pays 2:1\n"
			helpText += "- `range`: Bet on high(19-36)/low(1-18), pays 2:1\n"
			helpText += "- `dozen`: Bet on 1st/2nd/3rd dozen, pays 3:1\n"
			helpText += "- `column`: Bet on 1st/2nd/3rd column, pays 3:1\n\n"
			helpText += footer

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: helpText,
				},
			})
		},
		"balance": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

			// if balance is not yet set, set it to 0
			bal, err := db.Get(ctx, i.Member.User.ID).Result()
			if err != nil {
				if err != redis.Nil {
					log.Printf("Error getting balance: %v", err)
					return
				}
				db.Set(ctx, i.Member.User.ID, "150", 0)
				bal = "150"
			}

			balInt, err := strconv.Atoi(bal)
			if err != nil {
				log.Printf("Error converting balance to int: %v", err)
				return
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Your current balance is: \n# " + renderAmount(balInt) + "\n" + footer,
				},
			})
			return
		},
		"fountain": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Get last claim time
			lastClaimKey := fmt.Sprintf("fountain:%s", i.Member.User.ID)
			lastClaim, err := db.Get(ctx, lastClaimKey).Result()
			if err != nil && err != redis.Nil {
				log.Printf("Error getting last claim time: %v", err)
				return
			}

			// Check if user can claim
			if err != redis.Nil {
				lastClaimTime, err := strconv.ParseInt(lastClaim, 10, 64)
				if err != nil {
					log.Printf("Error parsing last claim time: %v", err)
					return
				}
				timeSinceLastClaim := time.Now().Unix() - lastClaimTime
				if timeSinceLastClaim < 3600 { // 1 hour in seconds
					timeLeft := 3600 - timeSinceLastClaim
					minutes := timeLeft / 60
					seconds := timeLeft % 60
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("You need to wait %d minutes and %d seconds before claiming again.\n%s", minutes, seconds, footer),
						},
					})
					return
				}
			}

			// Get current balance
			bal, err := db.Get(ctx, i.Member.User.ID).Result()
			if err != nil && err != redis.Nil {
				log.Printf("Error getting balance: %v", err)
				return
			}
			if err == redis.Nil {
				db.Set(ctx, i.Member.User.ID, "150", 0)
				bal = "150"
			}

			// Add fountain coins
			balInt, err := strconv.Atoi(bal)
			if err != nil {
				log.Printf("Error converting balance to int: %v", err)
				return
			}
			balInt += 50

			// Update balance and last claim time
			db.Set(ctx, i.Member.User.ID, strconv.Itoa(balInt), 0)
			db.Set(ctx, lastClaimKey, strconv.FormatInt(time.Now().Unix(), 10), 0)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("You claimed 50 coins from the fountain!\nYour new balance is: %s\n%s", renderAmount(balInt), footer),
				},
			})
		},
		"blackjack": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Check if user already has a game in progress
			if game := blackjack.GetGame(i.Member.User.ID); game != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You already have a game in progress! Finish it first.\n" + footer,
					},
				})
				return
			}

			// Check if the user has enough balance
			bal, err := db.Get(ctx, i.Member.User.ID).Result()
			if err != nil {
				if err != redis.Nil {
					log.Printf("Error getting balance: %v", err)
					return
				}
				db.Set(ctx, i.Member.User.ID, "150", 0)
				bal = "150"
			}
			balInt, err := strconv.Atoi(bal)
			if err != nil {
				log.Printf("Error converting balance to int: %v", err)
				return
			}
			fmt.Printf("User %s has balance %d\n", i.Member.User.ID, balInt)
			bet := int(i.ApplicationCommandData().Options[0].IntValue())
			if bet > balInt {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You don't have enough balance to play this game. \n" + footer,
					},
				})
				return
			}

			balInt -= bet
			db.Set(ctx, i.Member.User.ID, strconv.Itoa(balInt), 0)

			game := blackjack.NewGame(i.Member.User.ID, int(i.ApplicationCommandData().Options[0].IntValue()))
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content:    game.Render() + footer,
					Components: game.RenderButtons(),
				},
			})
		},
		"roulette": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Check if the user has enough balance
			bal, err := db.Get(ctx, i.Member.User.ID).Result()
			if err != nil {
				if err != redis.Nil {
					log.Printf("Error getting balance: %v", err)
					return
				}
				db.Set(ctx, i.Member.User.ID, "150", 0)
				bal = "150"
			}
			balInt, err := strconv.Atoi(bal)
			if err != nil {
				log.Printf("Error converting balance to int: %v", err)
				return
			}
			bet := int(i.ApplicationCommandData().Options[0].IntValue())
			if bet > balInt {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You don't have enough balance to play this game. \n" + footer,
					},
				})
				return
			}

			balInt -= bet
			db.Set(ctx, i.Member.User.ID, strconv.Itoa(balInt), 0)

			betType := i.ApplicationCommandData().Options[1].StringValue()
			betValue := i.ApplicationCommandData().Options[2].StringValue()

			// Validate bet value based on type
			var validationError string
			switch betType {
			case "number":
				num, err := strconv.Atoi(betValue)
				if err != nil || num < 0 || num > 36 {
					validationError = "Please enter a number between 0 and 36"
				}
			case "color":
				if betValue != "red" && betValue != "black" && betValue != "green" {
					validationError = "Please enter a valid color: red, black, or green"
				}
			case "parity":
				if betValue != "even" && betValue != "odd" {
					validationError = "Please enter either 'even' or 'odd'"
				}
			case "range":
				if betValue != "high" && betValue != "low" {
					validationError = "Please enter either 'high' or 'low'"
				}
			case "dozen":
				dozen, err := strconv.Atoi(betValue)
				if err != nil || dozen < 1 || dozen > 3 {
					validationError = "Please enter a dozen number between 1 and 3"
				}
			case "column":
				column, err := strconv.Atoi(betValue)
				if err != nil || column < 1 || column > 3 {
					validationError = "Please enter a column number between 1 and 3"
				}
			default:
				validationError = "Invalid bet type"
			}

			if validationError != "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Error: %s\n%s", validationError, footer),
					},
				})
				return
			}

			// Generate random number between 0 and 36
			result := randGen.Intn(37)
			resultColor := "green"
			if result != 0 {
				if result%2 == 0 {
					resultColor = "black"
				} else {
					resultColor = "red"
				}
			}

			// Check if bet won
			won := false
			switch betType {
			case "number":
				num, err := strconv.Atoi(betValue)
				if err == nil && num == result {
					won = true
					balInt += bet * 35 // 35:1 payout for single number
				}
			case "color":
				if betValue == resultColor {
					won = true
					balInt += bet * 2 // 2:1 payout for color
				}
			case "parity":
				if (betValue == "even" && result%2 == 0 && result != 0) ||
					(betValue == "odd" && result%2 == 1) {
					won = true
					balInt += bet * 2 // 2:1 payout for even/odd
				}
			case "range":
				if (betValue == "high" && result >= 19) ||
					(betValue == "low" && result >= 1 && result <= 18) {
					won = true
					balInt += bet * 2 // 2:1 payout for high/low
				}
			case "dozen":
				dozen, err := strconv.Atoi(betValue)
				if err == nil && result > 0 {
					if (dozen == 1 && result <= 12) ||
						(dozen == 2 && result > 12 && result <= 24) ||
						(dozen == 3 && result > 24) {
						won = true
						balInt += bet * 3 // 3:1 payout for dozen
					}
				}
			case "column":
				column, err := strconv.Atoi(betValue)
				if err == nil && result > 0 {
					// Real roulette column mapping
					column1 := []int{3, 6, 9, 12, 15, 18, 21, 24, 27, 30, 33, 36}
					column2 := []int{2, 5, 8, 11, 14, 17, 20, 23, 26, 29, 32, 35}
					column3 := []int{1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34}

					var inColumn bool
					switch column {
					case 1:
						inColumn = contains(column1, result)
					case 2:
						inColumn = contains(column2, result)
					case 3:
						inColumn = contains(column3, result)
					}

					if inColumn {
						won = true
						balInt += bet * 3 // 3:1 payout for column
					}
				}
			}

			// Update balance
			db.Set(ctx, i.Member.User.ID, strconv.Itoa(balInt), 0)

			// Create wheel visualization
			wheel := "```\n"
			wheel += fmt.Sprintf("🎲 %d %s\n", result, resultColor)
			wheel += "┌─────────────────┐\n"
			wheel += "│  🟥⬛🟥⬛🟥⬛🟥  │\n"
			wheel += "│  ⬛🟥⬛🟥⬛🟥⬛  │\n"
			wheel += "│  🟥⬛🟥⬛🟥⬛🟥  │\n"
			wheel += "└─────────────────┘\n"
			wheel += "```"

			// Create result message
			resultMsg := fmt.Sprintf("You bet %s on %s\n", renderAmount(bet), betValue)
			if won {
				resultMsg += "🎉 **You won!**\n"
			} else {
				resultMsg += "😢 **You lost!**\n"
			}
			resultMsg += fmt.Sprintf("Your new balance is: %s\n%s", renderAmount(balInt), footer)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: wheel + "\n" + resultMsg,
				},
			})
			return
		},
	}

	componentHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"hit": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			game := blackjack.GetGame(i.Member.User.ID)
			if game == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "No game in progress. Start a new game with /blackjack\n" + footer,
					},
				})
				return
			}
			ok := game.Hit()
			if !ok {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseUpdateMessage,
					Data: &discordgo.InteractionResponseData{
						Content: "Invalid move! The game has ended.\n" + footer,
					},
				})
				return
			}
			buttons := game.RenderButtons()
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Content:    game.Render() + footer,
					Components: buttons,
				},
			})
		},
		"stand": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			game := blackjack.GetGame(i.Member.User.ID)
			if game == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "No game in progress. Start a new game with /blackjack\n" + footer,
					},
				})
				return
			}
			ok := game.Stand()
			if !ok {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseUpdateMessage,
					Data: &discordgo.InteractionResponseData{
						Content: "Invalid move! The game has ended.\n" + footer,
					},
				})
				return
			}
			buttons := game.RenderButtons()
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Content:    game.Render() + footer,
					Components: buttons,
				},
			})
		},
	}
)

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
			return
		}
		fmt.Printf("Identifier: %v\n", i.MessageComponentData().CustomID)
		h, ok := componentHandlers[i.MessageComponentData().CustomID]
		if !ok {
			log.Printf("Unknown component handler: %v", i.MessageComponentData().CustomID)
			return
		}
		h(s, i)
	})
}

func main() {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	log.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	if *RemoveCommands {
		log.Println("Removing commands...")
		// // We need to fetch the commands, since deleting requires the command ID.
		// // We are doing this from the returned commands on line 375, because using
		// // this will delete all the commands, which might not be desirable, so we
		// // are deleting only the commands that we added.
		// registeredCommands, err := s.ApplicationCommands(s.State.User.ID, *GuildID)
		// if err != nil {
		// 	log.Fatalf("Could not fetch registered commands: %v", err)
		// }

		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, "", v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	log.Println("Gracefully shutting down.")
}

func contains(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
