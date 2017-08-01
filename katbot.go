package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// KatBot Configuration
var (
	StartTime = time.Now()

	Prefix = "~"
	Color  = 0x009688
	Name   = "KatBot"
	Icons  = "https://kittyhacker101.tk/Static/KatBot"
	Footer = "This is the footer for the help command."
	Play   = "Discord Playing Status"

	// Your API Keys (required)
	Token     = "Discord API Token"
	Cat       = "Cat API Key"
	Weather   = "Wunderground API Key"
	GoogleKey = "Google Custom Search Key"
	GoogleCX  = "Google Custom Search CX"
)

// SearchResult contains the contents of returned JSON from the Urban Dictionary API
type SearchResult struct {
	Tags    []string
	Results []Results `json:"list"`
}

// Results contains data from SearchResult
type Results struct {
	Author     string
	Definition string
	Example    string
}

// GoogleResult contains the contents of returned JSON from the Google Custom Search API
type GoogleResult struct {
	Info struct {
		Results string `json:"totalResults"`
	} `json:"searchInformation"`
	Items []Items
}

// Items contains data from GoogleResult
type Items struct {
	Title string
	Link  string
}

// WeatherResult contains the contents of returned JSON from the Weather Underground API
type WeatherResult struct {
	Current struct {
		Display struct {
			Full string
		} `json:"display_location"`
		Weather  string
		Temp     string `json:"temperature_string"`
		Humid    string `json:"relative_humidity"`
		Wind     string `json:"wind_string"`
		Dewpoint string `json:"dewpoint_string"`
		Icon     string
	} `json:"current_observation"`
}

// This function is called when loading the bot
func init() {
	flag.StringVar(&Token, "t", Token, "Bot Token")
	flag.Parse()
}

// Bot startup/closing is handled here
func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("Unable to create discord session!")
		return
	}

	// Setup handlers.
	dg.AddHandler(messageCreate)
	dg.AddHandler(ready)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("Unable to open connection!")
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	fmt.Println("Connected to Discord servers!")
	// Set playing status
	s.UpdateStatus(0, Play)
}

// This is called on every message received
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	tr := &http.Transport{DisableKeepAlives: true}
	client := &http.Client{Transport: tr}

	// Do not respond to self
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Help embed
	help := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: Name + " Commands", IconURL: Icons + "/kat.png"},
		Color:  Color,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   Prefix + "help",
				Value:  "Display a list of commands.",
				Inline: true,
			},
			{
				Name:   Prefix + "info",
				Value:  "View " + Name + "'s active configuration and hardware.",
				Inline: true,
			},
			{
				Name:   Prefix + "cat",
				Value:  "Display a random cat picture.",
				Inline: true,
			},
			{
				Name:   Prefix + "rand",
				Value:  "Generate a random number between zero and the number you enter.",
				Inline: true,
			},
			{
				Name:   Prefix + "weather",
				Value:  "Get the weather for a zip code.",
				Inline: true,
			},
			{
				Name:   Prefix + "urbandict",
				Value:  "Search for something in the Urban Dictionary.",
				Inline: true,
			},
			{
				Name:   Prefix + "google",
				Value:  "Search for something in Google.",
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: Footer, IconURL: Icons + "/kat.png"},
	}
	if strings.HasPrefix(m.Content, Prefix+"help") {
		s.ChannelMessageSendEmbed(m.ChannelID, help)
	}

	if strings.HasPrefix(m.Content, Prefix+"info") {
		var ostring string
		out, err := exec.Command("/bin/uname", "-mrs").Output()
		if err != nil {
			ostring = runtime.GOOS + " " + runtime.GOARCH
		} else {
			ostring = string(out)
		}
		gate, err := s.Gateway()
		if err != nil {
			gate = "Unknown"
		}
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{Name: Name + " Info", IconURL: Icons + "/kat.png"},
			Color:  Color,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Playing Status",
					Value:  Play,
					Inline: true,
				},
				{
					Name:   "Icon Server",
					Value:  Icons,
					Inline: true,
				},
				{
					Name:   "Gateway",
					Value:  gate,
					Inline: true,
				},
				{
					Name:   "Uptime",
					Value:  time.Since(StartTime).String(),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{Text: Name + " is currently running " + ostring, IconURL: Icons + "/kat.png"},
		})
	}

	// Get google searches with Custom Search API
	if strings.HasPrefix(m.Content, Prefix+"google ") {
		resp, err := client.Get("https://www.googleapis.com/customsearch/v1?prettyPrint=false&key=" + GoogleKey + "&cx=" + GoogleCX + "&q=" + m.Content[len(Prefix)+7:])
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Unable to fetch search results!")
			fmt.Println("[Warning] : Custom Search API Error")
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Unable to download search results!")
				fmt.Println("[Warning] : IO Error")
			} else {
				res := &GoogleResult{}
				err = json.Unmarshal(body, res)
				if err != nil || len(res.Items) <= 5 {
					s.ChannelMessageSend(m.ChannelID, "Unable to find search results!")
					fmt.Println("[Warning] : Parsing Error")
				} else {
					s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
						Author: &discordgo.MessageEmbedAuthor{Name: "Google Results for " + m.Content[len(Prefix)+7:] + " (" + res.Info.Results + " Total Results)", IconURL: Icons + "/googl.png"},
						Color:  Color,
						Fields: []*discordgo.MessageEmbedField{
							{
								Name:   res.Items[0].Title,
								Value:  res.Items[0].Link,
								Inline: true,
							},
							{
								Name:   res.Items[1].Title,
								Value:  res.Items[1].Link,
								Inline: true,
							},
							{
								Name:   res.Items[2].Title,
								Value:  res.Items[2].Link,
								Inline: true,
							},
							{
								Name:   res.Items[3].Title,
								Value:  res.Items[3].Link,
								Inline: true,
							},
							{
								Name:   res.Items[4].Title,
								Value:  res.Items[4].Link,
								Inline: true,
							},
							{
								Name:   res.Items[5].Title,
								Value:  res.Items[5].Link,
								Inline: true,
							},
						},
						Footer: &discordgo.MessageEmbedFooter{Text: "Search results provided by Google Custom Search.", IconURL: Icons + "/googl.png"},
					})
					fmt.Println("[Info] : Search results for '" + m.Content[len(Prefix)+7:] + "' successfully to " + m.Author.Username + "(" + m.Author.ID + ") in " + m.ChannelID)
				}
			}
		}
	}

	// Generate random numbers!
	if strings.HasPrefix(m.Content, Prefix+"rand ") {
		input, err := strconv.Atoi(m.Content[len(Prefix)+5:])
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Input invalid!")
			fmt.Println("[Warning] : Parsing Error")
		} else {
			s.ChannelMessageSend(m.ChannelID, strconv.Itoa(rand.Intn(input)))
		}
	}

	// Get weather reports from Wunderground
	if strings.HasPrefix(m.Content, Prefix+"weather ") {
		resp, err := client.Get("https://api.wunderground.com/api/" + Weather + "/conditions/q/" + m.Content[len(Prefix)+8:] + ".json")
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Unable to fetch weather report!")
			fmt.Println("[Warning] : Wunderground API Error")
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Unable to download weather report!")
				fmt.Println("[Warning] : IO Error")
			} else {
				res := &WeatherResult{}
				err = json.Unmarshal(body, res)
				if err != nil || res.Current.Weather == "" {
					s.ChannelMessageSend(m.ChannelID, "Unable to find location!")
					fmt.Println("[Warning] : Parsing Error")
				} else {
					s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
						Author: &discordgo.MessageEmbedAuthor{Name: "Weather for " + res.Current.Display.Full, IconURL: Icons + "/" + res.Current.Icon + ".png"},
						Color:  Color,
						Fields: []*discordgo.MessageEmbedField{
							{
								Name:   "Sky Conditions",
								Value:  res.Current.Weather,
								Inline: true,
							},
							{
								Name:   "Temp",
								Value:  res.Current.Temp,
								Inline: true,
							},
							{
								Name:   "Humidity",
								Value:  res.Current.Humid,
								Inline: true,
							},
							{
								Name:   "Dew Point",
								Value:  res.Current.Dewpoint,
								Inline: true,
							},
							{
								Name:   "Wind",
								Value:  res.Current.Wind,
								Inline: true,
							},
						},
						Footer: &discordgo.MessageEmbedFooter{Text: "Weather data provided by Wunderground API.", IconURL: Icons + "/wu.png"},
					})
					fmt.Println("[Info] : Weather for '" + res.Current.Display.Full + "' sent successfully to " + m.Author.Username + "(" + m.Author.ID + ") in " + m.ChannelID)
				}
			}
		}
	}

	// Fetch definition from Urban Dictionary.
	if strings.HasPrefix(m.Content, Prefix+"urbandict ") {
		resp, err := client.Get("https://api.urbandictionary.com/v0/define?term=" + m.Content[len(Prefix)+10:])
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Unable to fetch definition!")
			fmt.Println("[Warning] : Urban Dictionary API Error")
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Unable to download definition!")
				fmt.Println("[Warning] : IO Error")
			} else {
				res := &SearchResult{}
				err = json.Unmarshal(body, res)
				if err != nil || len(res.Results) <= 0 {
					s.ChannelMessageSend(m.ChannelID, "Unable to find definition!")
					fmt.Println("[Warning] : Parsing Error")
				} else {
					s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
						Author: &discordgo.MessageEmbedAuthor{Name: "Definition for " + m.Content[len(Prefix)+10:], IconURL: Icons + "/ud.png"},
						Color:  Color,
						Fields: []*discordgo.MessageEmbedField{
							{
								Name:   "Tags",
								Value:  strings.Trim(fmt.Sprintf("%v", res.Tags), "[]"),
								Inline: true,
							},
							{
								Name:   "Definition",
								Value:  res.Results[0].Definition,
								Inline: true,
							},
							{
								Name:   "Example",
								Value:  res.Results[0].Example,
								Inline: true,
							},
							{
								Name:   "Writer",
								Value:  res.Results[0].Author,
								Inline: true,
							},
						},
						Footer: &discordgo.MessageEmbedFooter{Text: "Dictionary data provided by Urban Dictionary.", IconURL: Icons + "/ud.png"},
					})
					fmt.Println("[Info] : Definition for '" + m.Content[len(Prefix)+10:] + "' sent successfully to " + m.Author.Username + "(" + m.Author.ID + ") in " + m.ChannelID)
				}
			}
		}
	}

	// Get pictures with the Cat API
	if strings.HasPrefix(m.Content, Prefix+"cat") {
		resp, err := client.Get("http://thecatapi.com/api/images/get?api_key=" + Cat + "&format=src")
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Unable to fetch cat!")
			fmt.Println("[Warning] : Cat API Error")
		} else {
			s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Author: &discordgo.MessageEmbedAuthor{Name: "Cat Picture", IconURL: Icons + "/cat.png"},
				Color:  Color,
				Image: &discordgo.MessageEmbedImage{
					URL: resp.Request.URL.String(),
				},
				Footer: &discordgo.MessageEmbedFooter{Text: "Cat pictures provided by TheCatApi", IconURL: Icons + "/cat.png"},
			})
			fmt.Println("[Info] : Cat sent successfully to " + m.Author.Username + "(" + m.Author.ID + ") in " + m.ChannelID)
		}
	}
}
