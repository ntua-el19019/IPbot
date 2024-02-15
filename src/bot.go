
package main

import (
	// "errors"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"

	// "fmt"
	"log"
	"os"
	"os/signal"

	// 	"encoding/json"
	// 	"fmt"
	// 	"io"
	// 	"net/http"
	//     "errors"
	// 	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type YourStruct struct {
	IPAddress          string   `json:"ip_address"`
	City               string   `json:"city"`
	CityGeonameID      int      `json:"city_geoname_id"`
	Region             string   `json:"region"`
	RegionISOCode      string   `json:"region_iso_code"`
	RegionGeonameID    int      `json:"region_geoname_id"`
	PostalCode         string   `json:"postal_code"`
	Country            string   `json:"country"`
	CountryCode        string   `json:"country_code"`
	CountryGeonameID   int      `json:"country_geoname_id"`
	CountryIsEU        bool     `json:"country_is_eu"`
	Continent          string   `json:"continent"`
	ContinentCode      string   `json:"continent_code"`
	ContinentGeonameID int      `json:"continent_geoname_id"`
	Longitude          float64  `json:"longitude"`
	Latitude           float64  `json:"latitude"`
	Security           Security `json:"security"`
	Timezone           Timezone `json:"timezone"`
}

type Security struct {
	IsVPN bool `json:"is_vpn"`
}

type Timezone struct {
    Name         string `json:"timezone.name"`
    Abbreviation string `json:"abbreviation"`
    GMTOffset    int    `json:"gmt_offset"`
    CurrentTime  string `json:"timezone.current_time"`
    IsDST        bool   `json:"is_dst"`
}

// Bot parameters
var (
	GuildID        = flag.String("guild", "", "Test guild ID. If not passed - bot registers commands globally")
	BotToken       = flag.String("token", "", "Bot access token")
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
)

var s *discordgo.Session

func init() { flag.Parse() }

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	
}

var (
	// dmPermission                   = false
	// defaultMemberPermissions int64 = discordgo.PermissionManageServer

	commands = []*discordgo.ApplicationCommand{
        {
            Name:        "findme",
            Description: "Fetches and displays your IP information",
            Options: []*discordgo.ApplicationCommandOption{
                {
                    Type:        discordgo.ApplicationCommandOptionString,
                    Name:        "ip",
                    Description: "The IP to fetch information for",
                    Required:    true,
                },
            },
        },
    }
    

		// {
		// 	Name:                     "permission-overview",
		// 	Description:              "Command for demonstration of default command permissions",
		// 	DefaultMemberPermissions: &defaultMemberPermissions,
		// 	DMPermission:             &dmPermission,
		// },

		
	

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"findme": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

			godotenv.Load()
			// Get the API key from the environment
			authtoken := os.Getenv("API_KEY")
			
          
            // Get the IP from the command options
            ip := i.ApplicationCommandData().Options[0].Value.(string)

        
			apilink := "https://ipgeolocation.abstractapi.com/v1/?api_key="+authtoken+"&ip_address="
            // Put the IP in the URL
            url := fmt.Sprintf(apilink+"%s", ip)
            
            resp, err := http.Get(url)
            
			if err != nil {
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "Failed to fetch data",
				})
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "Failed to read response",
				})
				return
			}

			var specific YourStruct
			if err = json.Unmarshal(body, &specific); err != nil {
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "Failed to parse specific",
				})
				return
			}
			// var timezone Timezone
			// if err = json.Unmarshal(body, &specific); err != nil {
			// 	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			// 		Content: "Failed to parse timezone",
			// 	})
			// 	return
			// }

			var security Security
			if err = json.Unmarshal(body, &specific); err != nil {
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "Failed to parse security",
				})
				return
			}

			response := fmt.Sprintf("IP: %s, isVPN: %t, Country: %s, City: %s", specific.IPAddress, security.IsVPN, specific.Country, specific.City)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: response,
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
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, *GuildID, v)
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
			err := s.ApplicationCommandDelete(s.State.User.ID, *GuildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	log.Println("Gracefully shutting down.")
}
