package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"math"

	"strconv"
	"golang.org/x/exp/slices"
	"github.com/dotabuff/manta"
	"github.com/dotabuff/manta/dota"
)


type HeroPlayer struct {
	HeroID   int32
	PlayerID uint64
}
type Glyph struct {
	MatchID     int
	Username    string
	UserSteamID string
	Minute      uint32
	Second      uint32
	Team        uint64  // Radiant team is 2 and dire team is 3
	HeroID      int32   // ID of hero (https://liquipedia.net/dota2/MediaWiki:Dota2webapi-heroes.json)
}

func oldParse(){
	log.Printf("Open file\n")
	// Create a new parser instance from a file. Alternatively see NewParser([]byte)
	f, err := os.Open("7941888611.dem")
	if err != nil {
		log.Fatalf("unable to open file: %s", err)
	}
	defer f.Close()

	p, err := manta.NewStreamParser(f)
	if err != nil {
		log.Fatalf("unable to create parser: %s", err)
	}

	// Register a callback, this time for the OnCUserMessageSayText2 event.
	//
	//func (c *Callbacks) OnCDOTAUserMsg_ChatMessage(fn func(*dota.CDOTAUserMsg_ChatMessage) error) {
	p.Callbacks.OnCDOTAUserMsg_ChatMessage(func(m *dota.CDOTAUserMsg_ChatMessage) error {
		log.Printf(strings.Repeat("#", 10))
		log.Printf("%s", m.GetMessageText())
		log.Printf("%s", m.ProtoReflect().Descriptor())
		log.Printf("%s", m.String())
		log.Printf("%d", m.GetSourcePlayerId())
		log.Printf(strings.Repeat("#", 10))

		return nil
	})
	p.Callbacks.OnCUserMessageSayText(func(m *dota.CUserMessageSayText) error {
		log.Printf("in callback saytext")
		return nil
	})
	p.Callbacks.OnCUserMessageSayText2(func(m *dota.CUserMessageSayText2) error {
		log.Printf("%s said: %s\n", m.GetParam1(), m.GetParam2())
		return nil
	})

	// Start parsing the replay!
	p.Start()

	log.Printf("Parse Complete!\n")

}


func GetGlyphsFromDem() ([]Glyph, error) {
	matchID:=7985118878
	filename := fmt.Sprintf("%d.dem", matchID)
	// Open file to parse
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	// Handle defer errors
	defer func(f *os.File) {
		if tempErr := f.Close(); tempErr != nil {
			err = tempErr
		}
	}(f)
	// Create stream parser
	p, err := manta.NewStreamParser(f)
	if err != nil {
		return nil, err
	}
	defer p.Stop()
	// Declare some variables for parsing
	var (
		gameCurrentTime, gameStartTime float64
		gamePaused                     bool
		pauseStartTick                 int32
		totalPausedTicks               int32

		heroPlayers = make([]HeroPlayer, 10)
		glyphs      []Glyph
		glyph       Glyph

		magicTime = 1100.0 // Time when heroes loaded TODO
	)

	p.Callbacks.OnCDOTAUserMsg_SpectatorPlayerUnitOrders(func(m *dota.CDOTAUserMsg_SpectatorPlayerUnitOrders) error {
		if m.GetOrderType() == int32(dota.DotaunitorderT_DOTA_UNIT_ORDER_GLYPH) {
			entity := p.FindEntity(m.GetEntindex())
			glyph = Glyph{
				MatchID:     matchID,
				Username:    entity.Get("m_iszPlayerName").(string),
				UserSteamID: strconv.FormatInt(int64(entity.Get("m_steamID").(uint64)), 10),
				Minute:      uint32(gameCurrentTime-gameStartTime) / 60,
				Second:      uint32(math.Round(gameCurrentTime-gameStartTime)) % 60,
				Team:        entity.Get("m_iTeamNum").(uint64),
			}
			if !slices.Contains(glyphs, glyph) {
				glyphs = append(glyphs, glyph)
			}
		}
		return nil
	})
	p.OnEntity(func(e *manta.Entity, op manta.EntityOp) error {
		switch e.GetClassName() {
		case "CDOTAGamerulesProxy":
			gameStartTime = float64(e.Get("m_pGameRules.m_flGameStartTime").(float32))
			gamePaused = e.Get("m_pGameRules.m_bGamePaused").(bool)
			pauseStartTick = e.Get("m_pGameRules.m_nPauseStartTick").(int32)
			totalPausedTicks = e.Get("m_pGameRules.m_nTotalPausedTicks").(int32)
			if gamePaused {
				gameCurrentTime = float64((pauseStartTick - totalPausedTicks) / 30)
			} else {
				gameCurrentTime = float64((int32(p.NetTick) - totalPausedTicks) / 30)
			}
		case "CDOTA_PlayerResource":
			if gameCurrentTime < magicTime {
				for i := 0; i < 10; i++ {
					heroPlayers[i].HeroID, _ = e.GetInt32("m_vecPlayerTeamData.000" + strconv.Itoa(i) + ".m_nSelectedHeroID")
					heroPlayers[i].PlayerID, _ = e.GetUint64("m_vecPlayerData.000" + strconv.Itoa(i) + ".m_iPlayerSteamID")
				}
			}
		}
		return nil
	})

	if err = p.Start(); err != nil {
		return nil, err
	}

	for k := range glyphs {
		for l := range heroPlayers {
			if glyphs[k].UserSteamID == strconv.FormatInt(int64(heroPlayers[l].PlayerID), 10) {
				glyphs[k].HeroID = heroPlayers[l].HeroID
				break
			}
		}
	}
	return glyphs, err
}

func main() {

	glyphs, err := GetGlyphsFromDem()
	fmt.Printf("matchID: %d\n", glyphs[0].MatchID)
	for i := range glyphs{
		fmt.Printf("Username: %s ", glyphs[i].Username)
		fmt.Printf("Minute: %d ", glyphs[i].Minute)
		fmt.Printf("Second: %d\n", glyphs[i].Second)
	}
	if err != nil{
		log.Fatal(err)
	}
}
