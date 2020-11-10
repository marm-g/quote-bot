package main

import (
    "fmt"
    "io/ioutil"
    "strings"
    "syscall"
    "strconv"
    "os"
    "os/signal"
    "github.com/bwmarrin/discordgo"
)

const prefix string = "!"
const quoteFile string = "./quotes.txt"
var nextQuoteID = 0

func check(e error) {
    if e != nil {
        panic(e)
    }
}

func contains (source []string, target string) bool {
    for _, str := range source {
        if str == target {
            return true
        }
    }
    return false
}

func getInitialNextQuoteID() int {
    quotes := getQuotes()
    return len(quotes)
}

func getQuotes() []string {
    quotes, err := ioutil.ReadFile(quoteFile)
    check(err)
    splitQuotes := strings.Split(string(quotes), "\n");
    // the last index is just blank (after the final newline) so we cut it off
    return splitQuotes[:len(splitQuotes)-1]
}

func getSpecificQuote(id int) string {
    id = id - 1
    quotes := getQuotes()
    if id >= len(quotes) || id < 0 {
        return "Quote doesn't exist."
    }
    return quotes[id]
}


func addQuote(quote string) bool {
    quotes, err := os.OpenFile(quoteFile, os.O_WRONLY | os.O_APPEND, 0644)
    if (err != nil) {
        return false
    }
    _, writeErr := quotes.WriteString(quote + "\n")
    if (writeErr != nil) {
        return false
    }
    nextQuoteID = nextQuoteID + 1
    return true
}

func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
    if message.Author.ID == session.State.User.ID {
        return
    }

    content := message.Content

    if strings.HasPrefix(content, prefix) {
        commands := []string{"addquote", "quote"}
        contentWithoutPrefix := strings.Replace(content, prefix, "", 1)
        splitContent := strings.Split(contentWithoutPrefix, " ")
        command := splitContent[0]
        if contains(commands, command) {
            splitContent = strings.SplitAfter(contentWithoutPrefix, command + " ")
            // actually do the command
            switch command {
            case "addquote":
                if len(splitContent) < 2 {
                    session.ChannelMessageSend(message.ChannelID, "No arguments provided. Usage: `addquote <quote>`")
                    return
                }
                args := splitContent[1]
                // TODO: strip newlines from quotes
                result := addQuote(strings.ReplaceAll(args, "\n", " "))
                if result == true {
                    session.ChannelMessageSend(message.ChannelID, "New quote added!")
                    return
                } else {
                    session.ChannelMessageSend(message.ChannelID, "Failed to add quote. Try again, or yell at Reno.")
                }
                return
            case "quote":
                if len(splitContent) < 2 {
                    session.ChannelMessageSend(message.ChannelID, "No arguments provided. Usage: `quote <quote number>`")
                    return
                }
                args := splitContent[1]
                quoteID, err := strconv.Atoi(args)
                if (err != nil) {
                    quoteID = 0 
                }
                quote := getSpecificQuote(quoteID)
                session.ChannelMessageSend(message.ChannelID, quote)
                return
            }
            session.ChannelMessageSend(message.ChannelID, message.Content)
        }
    }
}

func main () {
    // for _safety_  
    data, err := ioutil.ReadFile("./.token")
    check(err)
    token := string(data)
    token = strings.Trim(token, "\n")

    discord, err := discordgo.New("Bot " + token)
    check(err)

    discord.AddHandler(messageCreate)

    err = discord.Open()
    check(err)

    nextQuoteID = getInitialNextQuoteID()

    fmt.Println("Bot running. CTRL-C to stop.")
    // create a channel that looks for system signals (that has a size of one event)
    osChannel := make(chan os.Signal, 1)
    signal.Notify(osChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
    // TODO: what the heck does this do
    <-osChannel

    discord.Close()
}
