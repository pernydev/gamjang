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

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name: "balance",
			// All commands and options must have a description
			// Commands/options without description will fail the registration
			// of the command.
			Description: "Check your balance",
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
				db.Set(ctx, i.Member.User.ID, 0, 0)
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
		},
	}
)

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
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
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, "1356370074344230912", v)
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
			err := s.ApplicationCommandDelete(s.State.User.ID, "1356370074344230912", v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	log.Println("Gracefully shutting down.")
}
