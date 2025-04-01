package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"

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
	minBet   = float64(1)
	commands = []*discordgo.ApplicationCommand{
		{
			Name: "balance",
			// All commands and options must have a description
			// Commands/options without description will fail the registration
			// of the command.
			Description: "Check your balance",
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
					Name:        "color",
					Description: "Bet color",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "Red",
							Value: "red",
						},
						{
							Name:  "Black",
							Value: "black",
						},
						{
							Name:  "Green",
							Value: "green",
						},
					},
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
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
		"blackjack": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

			value := ""
			color := i.ApplicationCommandData().Options[1].StringValue()
			switch color {
			case "red":
				value = "🟥⬛🟥⬛🟥⬛🟥"
			case "black":
				value = "⬛🟥⬛🟥⬛🟥⬛"
			case "green":
				value = "🟥⬛🟥⬛🟥⬛🟥"
			default:
				value = "?"

			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You bet " + renderAmount(bet) + " on " + color + "\n" +
						"Spinning the wheel...\n\n" +
						"**Result:**\n# " +
						value + "\n" +
						"# ▪️▪️▪️🔺▪️▪️▪️\n" +
						"**You lost!**\n" +
						footer,
				},
			})

			return
		},
	}

	componentHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"hit": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			game := blackjack.GetGame(i.Member.User.ID)
			if game == nil {
				return
			}
			ok := game.Hit()
			if !ok {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseUpdateMessage,
					Data: &discordgo.InteractionResponseData{
						Content: "```ansi\n\u001b[0;31mError 0x7065726E79: Quantum Entanglement Exception in Module 'HyperThreadedVoid'\n```\n" + footer,
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
				return
			}
			ok := game.Stand()
			if !ok {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseUpdateMessage,
					Data: &discordgo.InteractionResponseData{
						Content: "```ansi\n\u001b[0;31mError 0x7065726E79: Quantum Entanglement Exception in Module 'HyperThreadedVoid'\n```\n" + footer,
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
