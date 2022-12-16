package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
)

var RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type quoteRow struct {
	ID        int
	QuoteText string
	DateAdded int
}

var commandsHaveDMPermission = false
var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "quote",
			Description: "Retrieves a random quote, or a specific quote",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "quote-id",
					Description: "ID of specific quote",
					Type:        discordgo.ApplicationCommandOptionInteger,
					Required:    false,
				},
			},
			DMPermission: &commandsHaveDMPermission,
		},
		{
			Name:        "addquote",
			Description: "Adds a new quote to the database",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "quote-text",
					Description: "The text of the quote you want to enter",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
			DMPermission: &commandsHaveDMPermission,
		},
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"quote": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := formatSlashCommandOptions(i.ApplicationCommandData().Options)
			if quoteID, ok := options["quote-id"]; ok {
				// Getting a specific quote
				quote := getSpecificQuote(int(quoteID.IntValue()))
				if quote.ID > 0 {
					slashCommandTextRespond("> "+quote.QuoteText, s, i)
				} else {
					slashCommandTextRespond("Quote not found!", s, i)
				}
				return
			}
			quotes := getQuotes()
			quote := quotes[rand.Intn(len(quotes))]
			msg := "> " + quote.QuoteText + "\n Quote ID " + strconv.Itoa(quote.ID)
			slashCommandTextRespond(msg, s, i)
		},
		"addquote": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := formatSlashCommandOptions(i.ApplicationCommandData().Options)
			msg := "No quote text provided."
			if quoteText, ok := options["quote-text"]; ok {
				res := addQuote(getUsernameFromInteraction(i), quoteText.StringValue())
				msg = "Something went wrong adding a quote - try again later."
				if res > 0 {
					msg = "New quote added! Quote ID: " + strconv.FormatInt(res, 10)
				}
			}
			slashCommandTextRespond(msg, s, i)
		},
	}
)

func formatSlashCommandOptions(opts []*discordgo.ApplicationCommandInteractionDataOption) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(opts))
	for _, opt := range opts {
		optionMap[opt.Name] = opt
	}
	return optionMap
}

func slashCommandTextRespond(msg string, s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
}

func getUsernameFromInteraction(i *discordgo.InteractionCreate) string {
	if i.Member != nil {
		return i.Member.User.Username
	}
	if i.User != nil {
		return i.User.Username
	}
	return ""
}

func getQuotes() []quoteRow {
	db, err := sql.Open("sqlite3", "quotes.db")
	check(err)
	query := "SELECT id,quoteText,dateAdded FROM quote;"
	rows, dbErr := db.Query(query)
	check(dbErr)
	returnObject := make([]quoteRow, 0)
	for rows.Next() {
		var (
			id        int
			quoteText string
			dateAdded int
		)
		if err := rows.Scan(&id, &quoteText, &dateAdded); err != nil {
			log.Fatal(err)
		}
		returnObject = append(returnObject, quoteRow{id, quoteText, dateAdded})
	}
	return returnObject
}

func getSpecificQuote(targetId int) quoteRow {
	db, err := sql.Open("sqlite3", "quotes.db")
	check(err)
	row := db.QueryRow("SELECT id,quoteText,dateAdded FROM quote WHERE id=?", targetId)
	var (
		id        int
		quoteText string
		dateAdded int
	)
	if err := row.Scan(&id, &quoteText, &dateAdded); err != nil {
		// Indicates no row was found; return a blank row
		return quoteRow{}
	}
	return quoteRow{id, quoteText, dateAdded}
}

func addQuote(username string, quote string) int64 {
	date := strconv.FormatInt(time.Now().Unix(), 10)
	query := "INSERT INTO quote (username, quoteText, dateAdded) VALUES (?,?,?)"
	db, err := sql.Open("sqlite3", "quotes.db")
	if err != nil {
		log.Println("Couldn't connect to db")
		log.Fatal(err)
		return 0
	}
	result, execErr := db.Exec(query, username, quote, date)
	if execErr != nil {
		log.Println("Error exectuting query")
		log.Println(query)
		log.Fatal(execErr)
		return 0
	}
	newQuoteId, resErr := result.LastInsertId()
	if resErr != nil {
		log.Println("Error getting last insert id")
		log.Fatal(resErr)
		return 0
	}
	return newQuoteId
}

func interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if handler, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
		handler(s, i)
	}
}

func main() {
	token := os.Getenv("QB_OAUTH_TOKEN")
	if len(token) <= 0 {
		fmt.Println("The environment variable QB_OAUTH_TOKEN is required to be set before running the bot.")
		os.Exit(1)
	}

	s, err := discordgo.New("Bot " + token)
	check(err)

	s.AddHandler(interactionHandler)

	err = s.Open()
	check(err)

	log.Println("Bot running. CTRL-C to stop.")

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			registeredCommands[i] = cmd
		}
	}

	// create a channel that looks for system signals (that has a size of one event)
	osChannel := make(chan os.Signal, 1)
	signal.Notify(osChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	// This flushes the value inside of osChannel, if it has one - ensures that
	// the channel will have room to continue processing signals as they come in.
	// In this case, we're using signal.Notify to catch SIGINT, SIGTERM & Interrupt
	// signals, and we redirect them out to the program.
	<-osChannel

	s.Close()
}
