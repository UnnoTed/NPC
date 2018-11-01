package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/UnnoTed/horizontal"
	"github.com/asdine/storm"
	"github.com/bwmarrin/discordgo"
	av "github.com/cmckee-dev/go-alpha-vantage"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	avAPIKey = ""
	avc      *av.Client

	db *storm.DB

	token    = "" // perms: 347200
	stocks   map[stockName]*stock
	comandos = []string{
		"-- Lista de comandos --\n```",
		"!npc ibovespa",
		"!npc dolar",
		"!npc euro",
		"!npc cotacao",
		"!npc cotação",
		"!npc todos",
		"!npc d `d para dolar`",
		"!npc i `i para ibovespa`",
		"!npc e `e para euro`",
		"!npc dolar tabela=sim colorido=nao diario=sim max=1",
		"!npc d tabela=nao colorido=sim diario=nao max=5",
		"!npc euro diario",
		"!npc codigo fonte",
		"```",
	}
)

func main() {
	debug := os.Getenv("NPC_DEBUG") == "true"
	avAPIKey = os.Getenv("NPC_APIKEY")

	if debug {
		log.Logger = log.Output(horizontal.ConsoleWriter{Out: os.Stderr})
		log.Debug().Msg("Debug mode activaTed")
		log.Level(zerolog.DebugLevel)
		token = os.Getenv("NPC_TOKEN_DEBUG")

	} else {
		token = os.Getenv("NPC_TOKEN")
	}

	if token == "" {
		panic("Error: Missing Discord BOT's token")
	}

	if avAPIKey == "" {
		panic("Error: Missing AlphaVantage's api key")
	}

	avc = av.NewClient(avAPIKey)

	var err error
	db, err = storm.Open("npc.db")
	defer db.Close()
	if err != nil {
		panic(err)
	}

	// default stocks
	stocks = map[stockName]*stock{
		stockNameIbovespa: &stock{ID: stockNameIbovespa, Name: string(stockNameIbovespa), Code: "^BVSP"},
		stockNameDolar:    &stock{ID: stockNameDolar, Name: string(stockNameDolar), Code: "USDBRL=X"},
		stockNameEuro:     &stock{ID: stockNameEuro, Name: string(stockNameEuro), Code: "EURBRL=X"},
	}

	// get stocks from db
	var ss []*stock
	if err := db.All(&ss); err != nil {
		panic(err)
	}

	// replace stocks with the ones found in the local db
	for _, s := range ss {
		stocks[s.ID] = s
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(mensagem)

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// updates stocks each 50min
	go func() {
		t := time.NewTicker(50 * time.Minute)
		defer t.Stop()

		for {
			var status string
			for _, s := range stocks {
				log.Info().Msg("Getting stocks from " + s.Name)
				if err := s.get(); err != nil {
					log.Error().Err(err).Msg("Error getting " + s.Name)
					continue
				}

				time.Sleep(1 * time.Second)
				status += " " + s.status()
			}

			// set bot status: "Playing I=87000 D=3.87 E=4.15"
			if dg != nil && dg.StateEnabled {
				dg.UpdateStatus(0, status)
			}
			<-t.C
		}
	}()

	wait()
	dg.Close()
}

func wait() {
	fmt.Println("--------------------------\nPress CTRL+C to quit\n--------------------------")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func mensagem(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	msg := strings.ToLower(m.Content)
	if strings.HasPrefix(msg, "!npc ") {
		s.ChannelTyping(m.ChannelID)

		log.Debug().Str("Author", m.Author.Username).Str("command", msg).Msg("Message")
		cmds := strings.Split(msg, " ")

		if len(cmds) >= 2 {
			icfg := parseInfoConfig(cmds)

			var err error
			switch cmds[1] {
			case "codigo", "source":
				_, err = s.ChannelMessageSend(m.ChannelID, "https://github.com/UnnoTed/NPC")

			case "i", "ibovespa":
				info := stocks[stockNameIbovespa].info(icfg)
				_, err = s.ChannelMessageSend(m.ChannelID, info)

			case "d", "dolar":
				info := stocks[stockNameDolar].info(icfg)
				_, err = s.ChannelMessageSend(m.ChannelID, info)

			case "e", "euro":
				info := stocks[stockNameEuro].info(icfg)
				_, err = s.ChannelMessageSend(m.ChannelID, info)

			default:
				// sends all stocks to the chat
				all := []stockName{stockNameIbovespa, stockNameDolar, stockNameEuro}
				for _, id := range all {
					info := stocks[id].info(icfg)
					_, err = s.ChannelMessageSend(m.ChannelID, info)

					if err != nil {
						log.Error().Err(err).Msg("Error while trying to send message")
					}
				}

				err = nil
			}

			if err != nil {
				if strings.Contains(err.Error(), "Must be 2000 or fewer in length") {
					s.ChannelMessageSend(m.ChannelID, "Erro: texto muito longo para ser enviado.")
				}

				log.Error().Err(err).Msg("Error while trying to send message")
			}
		}

	} else if msg == "!npc" {
		log.Debug().Str("Author", m.Author.Username).Str("command", "!npc").Msg("Message")

		uch, err := s.UserChannelCreate(m.Author.ID)
		if err != nil {
			log.Error().Err(err).Msg("Error while trying to create a dm channel")
			return
		}

		s.ChannelMessageSend(uch.ID, strings.Join(comandos, "\n"))
	}
}
