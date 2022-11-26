package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
)

const prefix string = "!"

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func contains(source []string, target string) bool {
	for _, str := range source {
		if str == target {
			return true
		}
	}
	return false
}

type quoteRow struct {
	ID        int
	QuoteText string
	DateAdded int
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

func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == session.State.User.ID {
		return
	}

	content := message.Content

	if strings.HasPrefix(content, prefix) {
		commands := []string{"addquote", "quote", "addedby"}
		contentWithoutPrefix := strings.Replace(content, prefix, "", 1)
		splitContent := strings.Split(contentWithoutPrefix, " ")
		command := splitContent[0]
		if contains(commands, command) {
			splitContent = strings.SplitAfter(contentWithoutPrefix, command+" ")
			// actually do the command
			switch command {
			case "addquote":
				if len(splitContent) < 2 {
					session.ChannelMessageSend(message.ChannelID, "No arguments provided. Usage: `addquote <quote>`")
					return
				}
				args := splitContent[1]

				name := message.Author.Username
				result := addQuote(name, strings.ReplaceAll(args, "\n", " "))
				if result > 0 {
					session.ChannelMessageSend(message.ChannelID, "New quote added! Quote ID: "+strconv.FormatInt(result, 10))
					return
				} else {
					session.ChannelMessageSend(message.ChannelID, "Failed to add quote. Try again, or yell at Reno.")
				}
				return
			case "quote":
				msg := ""
				var quote quoteRow
				if len(splitContent) < 2 {
					// get a random quote
					quotes := getQuotes()
					quote = quotes[rand.Intn(len(quotes))]
					msg = "> " + quote.QuoteText
					msg += "\n Quote ID " + strconv.Itoa(quote.ID)
					// TODO: add a verbose flag that displays this
					// if quote.DateAdded > 0 {
					//	msg += " added on " + strconv.Itoa(quote.DateAdded)
					//}

				} else {
					args := splitContent[1]
					quoteID, err := strconv.Atoi(args)
					if err != nil {
						quoteID = 0
					}
					quote = getSpecificQuote(quoteID)
					if quote.ID <= 0 {
						msg = "Quote not found!"
					} else {
						msg = "> " + quote.QuoteText
					}
				}

				session.ChannelMessageSend(message.ChannelID, msg)
				return
			}
			session.ChannelMessageSend(message.ChannelID, message.Content)
		}
	}
}

func main() {
	// Get the bot's OAuth token
	data, err := os.ReadFile("./.token")
	check(err)
	token := string(data)
	token = strings.Trim(token, "\n")

	discord, err := discordgo.New("Bot " + token)
	check(err)

	discord.AddHandler(messageCreate)

	err = discord.Open()
	check(err)

	fmt.Println("Bot running. CTRL-C to stop.")
	// create a channel that looks for system signals (that has a size of one event)
	osChannel := make(chan os.Signal, 1)
	signal.Notify(osChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	// TODO: what the heck does this do
	<-osChannel

	discord.Close()
}
