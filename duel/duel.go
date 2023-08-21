package duel

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	//xwidget "fyne.io/x/fyne/widget"
	"github.com/civilware/Gnomon/structures"
	dreams "github.com/dReam-dApps/dReams"
	"github.com/dReam-dApps/dReams/dwidget"
	"github.com/dReam-dApps/dReams/menu"
	"github.com/dReam-dApps/dReams/rpc"
	"github.com/sirupsen/logrus"
)

type entries struct {
	Total int
	Index map[uint64]entry
	sync.RWMutex
}

type entry struct {
	Num       string     `json:"num"`
	Init      uint64     `json:"init"`
	Stamp     int64      `json:"stamp"`
	Ready     uint64     `json:"ready"`
	Items     uint64     `json:"items"`
	Rule      string     `json:"rule"`
	DM        string     `json:"dm"`
	Token     string     `json:"token"`
	Amt       uint64     `json:"amt"`
	Displayed bool       `json:"displayed"`
	Complete  bool       `json:"complete"`
	Height    int64      `json:"height"`
	Winner    string     `json:"winner"`
	Odds      uint64     `json:"odds"`
	Duelist   playerInfo `json:"duelist"`
	Opponent  playerInfo `json:"opponent"`
}

type playerInfo struct {
	Address string `json:"owner"`
	Char    string `json:"char"`
	Item1   string `json:"item1"`
	Item2   string `json:"item2"`
	Opt     uint64 `json:"option"`
	Value   uint64 `json:"value"`
	Died    bool   `json:"died"`
	Icon    icons  `json:"icons"`
}

type icons struct {
	Char  []byte `json:"char"`
	Item1 []byte `json:"item1"`
	Item2 []byte `json:"item2"`
}

type record struct {
	Win  uint64
	Loss uint64
}

type records struct {
	address string
	value   record
}

type leaderBoard struct {
	board []records
	list  *widget.List
	sync.RWMutex
}

var D dreams.ContainerStack
var logger = structures.Logger.WithFields(logrus.Fields{})
var Duels entries
var Leaders leaderBoard
var Joins dwidget.Lists
var Ready dwidget.Lists
var Finals dwidget.Lists

// Menu intro for dReams app
func DreamsMenuIntro() (entries map[string][]string) {
	entries = map[string][]string{
		"Asset Duels": {
			"PvP duels for Dero, tokens and assets",
			"Duels are a custom version of over under where the contract (Arena) determines outcome",
			"Character and item ranks determine odds for winners and losers payout",
			"Leader board for tracking player wins and losses",
			"Ref Service for automated Duel Refereeing",
			"Collections",
			"Game Modes",
			"How to play"},

		"Collections": {"Dero Desperados", "Desperados Guns", "More to come..."},

		"Game Modes": {"Death match", "Hardcore"},

		"Death match": {
			"Winner gets the losers items",
			"Loosing character goes to graveyard",
			"Characters can be revived from graveyard by anyone",
			"Revival fee is determined by Duel amount and discounted 20% each week it spends in the graveyard",
			"A death matches can be combined with hardcore mode"},

		"Hardcore": {
			"In hardcore mode all character and item ranks do not apply and there will be no odds, winner takes all",
			"There is no Ref, when a opponent joins, the Duel will be completed in that TX",
			"Hardcore mode can be combined with a death match"},

		"How to play": {
			"Connect to your Dero wallet and daemon",
			"Any Duel assets will populate in the drop down menus",
			"Select your character and items you'd like to use for this Duel",
			"Select amount, token and any game modes you'd like",
			"Once a Duel has been started it will show up in the joins list for ANYONE to join",
			"Once a opponent joins the duel will be ready for a referee (*excluding hardcore mode*)",
			"If a higher ranked opponent joins your duel you will get paid odds on any loss (*excluding hardcore mode*)",
			"The referee will complete the duel, paying out any odds",
			"If death match was selected the winner will be send the losers items",
			"Track your stats on the leader board"},
	}

	return
}

// Main duel process
func fetch(d *dreams.AppObject, disconnect func()) {
	Graveyard.Index = make(map[uint64]grave)
	Inventory.characters = make(map[string]asset)
	Inventory.items = make(map[string]asset)
	time.Sleep(3 * time.Second)
	var offset int
	var synced bool
	for {
		select {
		case <-d.Receive():
			if !rpc.Wallet.IsConnected() || !rpc.Daemon.IsConnected() {
				disconnect()
				Disconnected()
				Inventory.Character.ClearAll()
				Inventory.Item1.ClearAll()
				Inventory.Item2.ClearAll()
				synced = false
				d.WorkDone()
				continue
			}

			if !synced && menu.GnomonScan(d.IsConfiguring()) {
				logger.Println("[Duels] Syncing")
				Duels = getIndex()
				synced = true
			}

			if synced {
				if GetJoins() {
					Joins.List.Refresh()
				}

				if GetAllDuels() {
					Ready.List.Refresh()
				}

				if GetFinals() {
					Finals.List.Refresh()
				}

				GetGraveyard()
				if offset%10 == 0 {
					if !menu.Gnomes.IsClosing() {
						storeIndex()
					}
				}
			}

			D.LeftLabel.SetText(fmt.Sprintf("Total Duels Held: (%d)      Ready Duels: (%d)", Duels.Total, len(Ready.All)))
			D.RightLabel.SetText("dReams Balance: " + rpc.DisplayBalance("dReams") + "      Dero Balance: " + rpc.DisplayBalance("Dero") + "      Height: " + rpc.Wallet.Display.Height)

			offset++
			if offset > 10 {
				offset = 0
			}

			d.WorkDone()
		case <-d.CloseDapp():
			logger.Println("[Duels] Done")
			return
		}
	}
}

// Format duration into day, hour, minute, second string
func formatDuration(duration time.Duration) (dur string) {
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	switch days {
	case 0:
		// nothing
	case 1:
		dur = fmt.Sprintf("%d day, ", days)
	default:
		dur = fmt.Sprintf("%d days, ", days)
	}

	dur = dur + fmt.Sprintf("%d hours, %d minutes, %d seconds", hours, minutes, seconds)

	return
}

// Chop address for display
func chopAddr(addr string) string {
	if len(addr) < 66 {
		return ""
	}
	return addr[:10] + "..." + addr[56:]
}

// Get leader board record by address
func (lb *leaderBoard) getRecordByAddress(addr string) (record string) {
	for _, r := range lb.board {
		if r.address == addr {
			return fmt.Sprintf("(%dW - %dL)", r.value.Win, r.value.Loss)
		}
	}

	return
}

// Get leader board record by index
func (lb *leaderBoard) getRecordByIndex(i int) (record string) {
	return fmt.Sprintf("(%dW - %dL)", lb.board[i].value.Win, lb.board[i].value.Loss)
}

// Write a duel entry to map
func (d *entries) WriteEntry(i uint64, entry entry) {
	d.Lock()
	d.Index[i] = entry
	d.Unlock()
}

// Return a single duel entry from map
func (d *entries) SingleEntry(i uint64) (entry entry) {
	d.RLock()
	defer d.RUnlock()

	return d.Index[i]
}

// Gets joinable duels
func GetJoins() (update bool) {
	if menu.Gnomes.IsReady() {
		_, initValue := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "init")
		if initValue != nil {
			if _, rounds := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "rds"); rounds != nil {
				Duels.Total = int(rounds[0])
			}

			u := uint64(0)
			for {
				u++
				if u > initValue[0] {
					break
				}

				if !rpc.Wallet.IsConnected() || !menu.Gnomes.IsReady() {
					break
				}

				e := Duels.SingleEntry(u)

				n := strconv.Itoa(int(u))
				_, init := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "init_"+n)
				if init == nil || init[0] == 0 {
					Duels.Lock()
					delete(Duels.Index, u)
					Duels.Unlock()
					Joins.RemoveIndex(u)
					Ready.RemoveIndex(u)
					Finals.RemoveIndex(u)
					continue
				}

				if e.Num == "" {
					logger.Debugln("[GetJoins] Making")

					_, buffer := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "stamp_"+n)
					if buffer == nil {
						logger.Debugf("[GetAllDuels] %s no start stamp\n", n)
						buffer = append(buffer, 0)
					}

					address, _ := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "own_a_"+n)
					if address == nil {
						logger.Debugf("[GetJoins] %s no address\n", n)
						continue
					}

					if address[0] != rpc.Wallet.Address && time.Now().Unix() <= int64(buffer[0]) {
						logger.Debugf("[GetJoins] %s in buffer\n", n)
						continue
					}

					_, items := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "items_"+n)
					if items == nil {
						logger.Debugf("[GetJoins] %s no items\n", n)
						continue
					}

					deathmatch := "No"
					_, dm := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "dm_"+n)
					if dm == nil {
						logger.Debugf("[GetJoins] %s no dm\n", n)
						continue
					}

					if dm[0] == 1 {
						deathmatch = "Yes"
					}

					hardcore := "No"
					_, rule := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "rule_"+n)
					if rule == nil {
						logger.Debugf("[GetJoins] %s no rule\n", n)
						continue
					}

					if rule[0] == 1 {
						hardcore = "Yes"
					}

					_, amt := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "amt_"+n)
					if amt == nil {
						logger.Debugf("[GetJoins] %s no amt\n", n)
						continue
					}

					_, option := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "op_a_"+n)
					if option == nil {
						logger.Debugf("[GetJoins] %s no optA\n", n)
						continue
					}

					charA, _ := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "ch_a_"+n)
					if charA == nil {
						logger.Debugf("[GetJoins] %s no charA\n", n)
						continue
					}

					charIcon, err := downloadBytes(charA[0])
					if err != nil {
						charIcon = resourceUnknownIconPng.StaticContent
						logger.Debugf("[GetJoins] %s charIconA %v\n", n, err)
					}

					token, _ := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "tkn_"+n)
					if token == nil {
						logger.Debugf("[GetJoins] %s no token\n", n)
						token = append(token, "")
					}

					var item1Str, item2Str string
					var item1Img, item2Img []byte
					if items[0] >= 1 {
						item1, _ := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "i1_a_"+n)
						if item1 == nil {
							logger.Debugf("[GetJoins] %s should be a item1\n", n)
							continue
						}

						item1Img, err = downloadBytes(item1[0])
						if err != nil {
							logger.Errorln("[GetJoins]", err)
							item1Img = resourceUnknownIconPng.StaticContent
						}
						item1Str = item1[0]
					}

					if items[0] == 2 {
						item2, _ := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "i2_a_"+n)
						if item2 == nil {
							logger.Debugf("[GetJoins] %s should be a item2\n", n)
							continue
						}

						item2Img, err = downloadBytes(item2[0])
						if err != nil {
							logger.Errorln("[GetJoins]", err)
							item2Img = resourceUnknownIconPng.StaticContent
						}
						item2Str = item2[0]
					}

					logger.Debugln("[GetJoins] Storing A", n)
					e = entry{
						Num:      n,
						Init:     initValue[0],
						Stamp:    int64(buffer[0]),
						Items:    items[0],
						Rule:     hardcore,
						DM:       deathmatch,
						Token:    token[0],
						Amt:      amt[0],
						Complete: false,
						Duelist: playerInfo{
							Address: address[0],
							Char:    charA[0],
							Item1:   item1Str,
							Item2:   item2Str,
							Opt:     option[0],
							Value:   0,
							Icon: icons{
								Char:  charIcon,
								Item1: item1Img,
								Item2: item2Img,
							},
						},
					}
					Duels.WriteEntry(u, e)
					update = true
					Joins.All = append(Joins.All, u)
				} else if e.Opponent.Icon.Char == nil && !Joins.Exists(u) {
					Joins.All = append(Joins.All, u)
					update = true
				}
			}
		}
	}

	Joins.SortIndex()
	Ready.SortIndex()

	return
}

// Gets already joined duels awaiting ref
func GetAllDuels() (update bool) {
	if menu.Gnomes.IsReady() {
		for u, v := range Duels.Index {
			if !rpc.Wallet.IsConnected() || !menu.Gnomes.IsReady() {
				break
			}

			if v.Opponent.Char != "" {
				if Ready.Exists(u) {
					logger.Debugf("[GetAllDuels] %d b Char already here\n", u)
				} else if !v.Complete {
					Ready.All = append(Ready.All, u)
					Joins.RemoveIndex(u)
					update = true
				}

				continue
			}

			n := strconv.Itoa(int(u))
			if _, init := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "init_"+n); init != nil {
				address, _ := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "own_b_"+n)
				if address == nil {
					logger.Debugf("[GetAllDuels] %s no address B\n", n)
					continue
				}

				_, ready_stamp := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "ready_"+n)
				if ready_stamp == nil {
					logger.Debugf("[GetAllDuels] %s no ready stamp\n", n)
					ready_stamp = append(ready_stamp, 0)
				}

				char, _ := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "ch_b_"+n)
				if char == nil {
					logger.Debugf("[GetAllDuels] %s no charB\n", n)
					continue
				}

				charIcon, err := downloadBytes(char[0])
				if err != nil {
					charIcon = resourceUnknownIconPng.StaticContent
					logger.Debugf("[GetAllDuels] %s charIconB %v\n", n, err)
				}

				_, option := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "op_b_"+n)
				if option == nil {
					logger.Debugf("[GetAllDuels] %s no optB\n", n)
					continue
				}

				_, valA := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "v_a_"+n)
				if valA == nil {
					logger.Debugf("[GetAllDuels] %s no valA\n", n)
					continue
				}

				_, valB := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "v_b_"+n)
				if valB == nil {
					logger.Debugf("[GetAllDuels] %s no valB\n", n)
					continue
				}

				var item1Str, item2Str string
				var item1Img, item2Img []byte
				if v.Items >= 1 && v.Opponent.Icon.Item1 == nil {
					item1, _ := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "i1_b_"+n)
					if item1 == nil {
						logger.Debugf("[GetAllDuels] %s should be a item1\n", n)
						continue
					}

					item1Img, err = downloadBytes(item1[0])
					if err != nil {
						logger.Debugln("[GetAllDuels]", err)
						item1Img = resourceUnknownIconPng.StaticContent
					}
					item1Str = item1[0]
				}

				if v.Items == 2 && v.Opponent.Icon.Item2 == nil {
					item2, _ := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "i2_b_"+n)
					if item2 == nil {
						logger.Debugf("[GetAllDuels] %s should be a item2\n", n)
						continue
					}

					item2Img, err = downloadBytes(item2[0])
					if err != nil {
						logger.Debugln("[GetAllDuels]", err)
						item2Img = resourceUnknownIconPng.StaticContent
					}
					item2Str = item2[0]
				}

				Ready.All = append(Ready.All, u)
				Joins.RemoveIndex(u)

				v.Init = init[0]
				v.Ready = ready_stamp[0]
				v.Duelist.Value = valA[0]
				v.Opponent = playerInfo{
					Address: address[0],
					Char:    char[0],
					Item1:   item1Str,
					Item2:   item2Str,
					Opt:     option[0],
					Value:   valB[0],
					Icon: icons{
						Char:  charIcon,
						Item1: item1Img,
						Item2: item2Img,
					},
				}

				logger.Debugln("[GetAllDuels] Storing B Info", u)
				update = true
				Duels.WriteEntry(u, v)
			} else {
				update = true
				Ready.RemoveIndex(u)
				Duels.Lock()
				delete(Duels.Index, u)
				Duels.Unlock()
			}
		}
	}

	logger.Debugln("[GetAllDuels] Joins:", len(Joins.All), Joins.All, "Ready:", len(Ready.All), Ready.All, "Finals:", len(Finals.All), Finals.All, "Update:", update)

	return
}

// Gets final duel results
func GetFinals() (update bool) {
	if menu.Gnomes.IsReady() {
		for u, v := range Duels.Index {
			if !rpc.Wallet.IsConnected() || !menu.Gnomes.IsReady() {
				break
			}

			n := strconv.Itoa(int(u))
			if !v.Complete {
				if final, _ := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "final_"+n); final != nil {
					if winner := strings.Split(final[0], "_"); len(winner) >= 3 {
						v.Winner = rpc.DeroAddressFromKey(winner[0])
						v.Odds = rpc.Uint64Type(winner[2])
						Ready.RemoveIndex(u)
						if v.Odds <= 950 {
							v.Complete = true
							v.Height = rpc.GetDaemonTx(winner[1]).Block_Height
							Finals.All = append(Finals.All, u)
							if v.DM == "Yes" {
								if v.Odds > 475 {
									v.Duelist.Died = true
								} else {
									v.Opponent.Died = true
								}
							}
						}

						update = true
						Duels.WriteEntry(u, v)

						continue

					}
					logger.Debugf("[GetFinals] %s invalid winner string\n", n)
				}
			} else if v.Complete && !Finals.Exists(u) {
				Finals.All = append(Finals.All, u)
				update = true
			}
		}
	}

	Finals.SortIndex()

	return
}

// Get asset name of duel
func (duel entry) assetName() (asset_name string) {
	asset_name = "DERO"
	if name := rpc.GetAssetSCIDName(duel.Token); name != "" {
		asset_name = name
	}

	return
}

// Check all assets used for duel are valid
func (duel entry) validateCollection() bool {
	characters := []string{"Dero Desperados", "TestChars"}
	items := []string{"Desperado Guns", "TestItems"}

	switch duel.Items {
	case 0:
		if coll, _ := menu.Gnomes.GetSCIDValuesByKey(duel.Duelist.Char, "collection"); coll != nil {
			for _, c := range characters {
				if coll[0] == c {
					return true
				}
			}
		}
	case 1:
		var validChar bool
		if coll, _ := menu.Gnomes.GetSCIDValuesByKey(duel.Duelist.Char, "collection"); coll != nil {
			for _, c := range characters {
				if coll[0] == c {
					validChar = true
					break
				}
			}
		}

		if validChar {
			if item1, _ := menu.Gnomes.GetSCIDValuesByKey(duel.Duelist.Item1, "collection"); item1 != nil {
				for _, c := range items {
					if item1[0] == c {
						return true
					}
				}
			}
		}
	case 2:
		var validChar bool
		if coll, _ := menu.Gnomes.GetSCIDValuesByKey(duel.Duelist.Char, "collection"); coll != nil {
			for _, c := range characters {
				if coll[0] == c {
					validChar = true
					break
				}
			}
		}

		if validChar {
			var validItem bool
			if item1, _ := menu.Gnomes.GetSCIDValuesByKey(duel.Duelist.Item1, "collection"); item1 != nil {
				for _, c := range items {
					if item1[0] == c {
						validItem = true
						break
					}
				}
			}

			if validItem {
				if item2, _ := menu.Gnomes.GetSCIDValuesByKey(duel.Duelist.Item1, "collection"); item2 != nil {
					for _, c := range items {
						if item2[0] == c {
							return true
						}
					}
				}
			}
		}
	default:
		// Nothing
	}

	return false
}

// Create the title header string for duel results
func (duel entry) resultsHeaderString() string {
	return fmt.Sprintf("Duel #%s   Pot: (%s %s)   Items: (%d)   Death Match: (%s)   Hardcore: (%s)   Block: (%d)", duel.Num, rpc.FromAtomic(duel.Amt*2, 5), duel.assetName(), duel.Items, duel.DM, duel.Rule, duel.Height)
}

// Check if connected wallet is duelist or opponent
func (duel entry) checkDuelAddresses() bool {
	if rpc.Wallet.Address == duel.Duelist.Address {
		return true
	} else if rpc.Wallet.Address == duel.Opponent.Address {
		return true
	}

	return false

}

// Find time since when duel has been ready
func (duel entry) readySince() string {
	if duel.Ready == 0 {
		return "Completed"
	}

	now := time.Now()
	ready := time.Unix(int64(duel.Ready), 0)

	return formatDuration(now.Sub(ready))
}

// Display string for duel hit results
func (duel entry) endedIn() string {
	suffix := "th shot"
	u := duel.Init - 1
	switch u {
	case 1:
		suffix = "st shot"
	case 2:
		suffix = "nd shot"
	case 3:
		suffix = "rd shot"
	}
	return strconv.FormatUint(u, 10) + suffix
}

// Find odds for duels where ranks are different
//   - Pass r as opponent rank
func (duel entry) diffOdds(r uint64) (perc uint64, rank1 uint64, diff uint64) {
	rank1, _ = duel.getTotalRanks()
	if r > rank1 {
		diff = r - rank1
	}

	switch duel.Items {
	case 0:
		perc = 100 - (10 * diff)
	case 1:
		perc = 100 - (5 * diff)
	case 2:
		perc = 100 - (4 * diff)
	default:
		logger.Errorln("[diffOdds] Err - processing items")
	}

	return
}

// Owner and ref function to run regular duels
func (duel entry) refDuel() (tx string) {
	if !checkOwnerAndRefs() {
		logger.Warnln("[refDuel] You are not the owner or a ref on this SCID")
		return
	}

	var winner rune
	var address string
	var odds uint64

	optA := duel.Duelist.Opt
	valA := duel.Duelist.Value
	finalA := valA - optA

	optB := duel.Opponent.Opt
	valB := duel.Opponent.Value
	finalB := valB - optB

	logger.Debugln("[refDuel] A:", finalA, "B:", finalB)

	if finalA == 5 && finalB == 5 {
		if optA > optB {
			// A wins
			winner = 'A'
			address = duel.Duelist.Address
			logger.Debugln("[refDuel] A Wins setting odds", odds)
		} else if optB > optA {
			// B wins
			winner = 'B'
			address = duel.Opponent.Address
			odds = 950
			logger.Debugln("[refDuel] B Wins setting odds", odds)
		} else {
			odds = 1500
			logger.Errorln("[refDuel] Err - determining winner")
			return
		}
	} else {
		if finalA >= 5 && (finalB > finalA || finalB < 5) {
			// A wins
			winner = 'A'
			address = duel.Duelist.Address
			logger.Debugln("[refDuel] A Wins setting odds", odds)
		} else if finalB >= 5 && (finalA > finalB || finalA < 5) {
			// B wins
			winner = 'B'
			address = duel.Opponent.Address
			odds = 950
			logger.Debugln("[refDuel] B Wins setting odds", odds)
		} else {
			odds = 1600
			logger.Errorln("[refDuel] Err - determining winner")
			return
		}
	}

	diff := uint64(0)
	rank1, rank2 := duel.getTotalRanks()
	if rank1 < rank2 && winner == 'B' {
		diff = rank2 - rank1
	} else if rank2 < rank1 && winner == 'A' {
		diff = rank1 - rank2
	}

	switch duel.Items {
	case 0:
		if winner == 'A' {
			odds += (95 * diff)
		} else {
			odds -= (95 * diff)
		}

	case 1:
		if winner == 'A' {
			odds += (47 * diff)
		} else {
			odds -= (47 * diff)
		}

	case 2:
		if winner == 'A' {
			odds += (38 * diff)
		} else {
			odds -= (38 * diff)
		}

	default:
		odds = 1700
		logger.Errorln("[refDuel] Err - processing items")
		return
	}
	logger.Println("[refDuel]", string(winner), "items:", duel.Items, "rank1:", rank1, "rank2:", rank2, "diff:", diff, "odds:", odds, address)

	tx = duel.ref(duel.Num, address, winner, odds)

	return
}

// Dry run for owner and ref function to run regular duels
func (duel entry) dryRefDuel() (payout string) {
	if !checkOwnerAndRefs() {
		logger.Warnln("[refDuel] You are not the owner or a ref on this SCID")
		return
	}

	var winner rune
	var address string
	var odds uint64

	optA := duel.Duelist.Opt
	valA := duel.Duelist.Value
	finalA := valA - optA

	optB := duel.Opponent.Opt
	valB := duel.Opponent.Value
	finalB := valB - optB

	logger.Debugln("[refDuel] A:", finalA, "B:", finalB)

	if finalA == 5 && finalB == 5 {
		if optA > optB {
			// A wins
			winner = 'A'
			address = duel.Duelist.Address
			logger.Debugln("[refDuel] A Wins setting odds", odds)
		} else if optB > optA {
			// B wins
			winner = 'B'
			address = duel.Opponent.Address
			odds = 950
			logger.Debugln("[refDuel] B Wins setting odds", odds)
		} else {
			odds = 1500
			logger.Errorln("[refDuel] Err - determining winner")
			return
		}
	} else {
		if finalA >= 5 && (finalB > finalA || finalB < 5) {
			// A wins
			winner = 'A'
			address = duel.Duelist.Address
			logger.Debugln("[refDuel] A Wins setting odds", odds)
		} else if finalB >= 5 && (finalA > finalB || finalA < 5) {
			// B wins
			winner = 'B'
			address = duel.Opponent.Address
			odds = 950
			logger.Debugln("[refDuel] B Wins setting odds", odds)
		} else {
			odds = 1600
			logger.Errorln("[refDuel] Err - determining winner")
			return
		}
	}

	diff := uint64(0)
	rank1, rank2 := duel.getTotalRanks()
	if rank1 < rank2 && winner == 'B' {
		diff = rank2 - rank1
	} else if rank2 < rank1 && winner == 'A' {
		diff = rank1 - rank2
	}

	switch duel.Items {
	case 0:
		if winner == 'A' {
			odds += (95 * diff)
		} else {
			odds -= (95 * diff)
		}

	case 1:
		if winner == 'A' {
			odds += (47 * diff)
		} else {
			odds -= (47 * diff)
		}

	case 2:
		if winner == 'A' {
			odds += (38 * diff)
		} else {
			odds -= (38 * diff)
		}

	default:
		odds = 1700
		logger.Errorln("[refDuel] Err - processing items")
		return
	}
	logger.Debugln("[refDuel]", string(winner), "items:", duel.Items, "rank1:", rank1, "rank2:", rank2, "diff:", diff, "odds:", odds, address)

	var amt uint64
	pot := duel.Amt * 2
	if odds > 475 {
		// B wins
		amt = odds * pot / 1000
	} else {
		// A wins
		o := uint64(950)
		amt = (o - odds) * pot / 1000
	}

	side := "Duelist"
	if odds > 475 {
		side = "Opponent"
	}

	return fmt.Sprintf("%s wins, Payout: (%s %s)", side, rpc.FromAtomic(amt, 5), duel.assetName())
}

// Find duelist and opponent earning from a duel
func (duel entry) findEarning() (a, b uint64) {
	pot := duel.Amt * 2
	odds := uint64(950)
	if duel.Odds > 475 {
		// B wins
		b = duel.Odds * pot / 1000
		a = (odds - duel.Odds) * pot / 1000
	} else {
		// A wins
		b = duel.Odds * pot / 1000
		a = (odds - duel.Odds) * pot / 1000
	}

	return
}

// Returns the ranks of duelist and opponent
func (duel entry) getTotalRanks() (r1 uint64, r2 uint64) {
	switch duel.Items {
	case 0:
		r1 = validateAssetRank(duel.Duelist.Char)
		r2 = validateAssetRank(duel.Opponent.Char)
	case 1:
		r1 = validateAssetRank(duel.Duelist.Char) + validateAssetRank(duel.Duelist.Item1)
		r2 = validateAssetRank(duel.Opponent.Char) + validateAssetRank(duel.Opponent.Item1)
	case 2:
		r1 = validateAssetRank(duel.Duelist.Char) + validateAssetRank(duel.Duelist.Item1) + validateAssetRank(duel.Duelist.Item2)
		r2 = validateAssetRank(duel.Opponent.Char) + validateAssetRank(duel.Opponent.Item1) + validateAssetRank(duel.Opponent.Item2)
	default:
		logger.Errorln("[getAssetRank] Err - getting ranks", "r1:", r1, "r2:", r2)
	}

	return
}

// Return rank of duelist
func (duel entry) getDuelistRank() (r1 uint64) {
	switch duel.Items {
	case 0:
		r1 = validateAssetRank(duel.Duelist.Char)
	case 1:
		r1 = validateAssetRank(duel.Duelist.Char) + validateAssetRank(duel.Duelist.Item1)
	case 2:
		r1 = validateAssetRank(duel.Duelist.Char) + validateAssetRank(duel.Duelist.Item1) + validateAssetRank(duel.Duelist.Item2)
	default:
		logger.Errorln("[getDuelistRank] Err - getting rank")
	}

	return
}

// Create a icon image for duel character or item
//   - size 0 for small icon and 1 for large
//   - img switch defines if character or item image is returned
func (duel playerInfo) IconImage(size, img int) fyne.CanvasObject {
	switch img {
	case 0:
		if size == 0 {
			return iconSmall(duel.Icon.Char, duel.Char, duel.Died)
		}

		return iconLarge(duel.Icon.Char, duel.Char)
	case 1:
		if size == 0 {
			return iconSmall(duel.Icon.Item1, duel.Item1, false)
		}

		return iconLarge(duel.Icon.Item1, duel.Item1)
	case 2:
		if size == 0 {
			return iconSmall(duel.Icon.Item2, duel.Item2, false)
		}

		return iconLarge(duel.Icon.Item2, duel.Item2)
	default:
		if size == 0 {
			return iconSmall(resourceUnknownIconPng.StaticContent, "", false)
		}

		return iconLarge(resourceUnknownIconPng.StaticContent, "")
	}
}

// Find rank of character
func (p playerInfo) getCharacterRank() uint64 {
	return validateAssetRank(p.Char)
}

// Find rank of item 1 or 2
//   - i of 0 return item1 and 1 returns item2
func (p playerInfo) getItemRank(i int) uint64 {
	if i == 1 {
		return validateAssetRank(p.Item2)
	}
	return validateAssetRank(p.Item1)
}

// Creates the rank display string
func (p playerInfo) getRankString() (str string) {
	str = fmt.Sprintf("{R%d}", validateAssetRank(p.Char))
	if p.Item1 != "" {
		str = str + fmt.Sprintf(" {R%d}", validateAssetRank(p.Item1))
	}

	if p.Item2 != "" {
		str = str + fmt.Sprintf(" {R%d}", validateAssetRank(p.Item2))
	}

	return
}

// Finds duel results and returns result string
func (p playerInfo) findDuelResult() (result string) {
	a := p.Value - p.Opt
	if a == 5 {
		result = " Hit Target  \n\n" + findTargetHit(p.Opt)
	} else if a > 5 {
		result = fmt.Sprintf("Under by %d ", a-5)
	} else {
		result = "Over Target"
	}
	return
}

// Returns hit target string
func findTargetHit(u uint64) string {
	switch u {
	case 0:
		return " Leg  "
	case 1:
		return " Arm  "
	case 2:
		return " Chest  "
	case 3:
		return " Neck  "
	case 4:
		return " Head  "
	default:
		return ""
	}
}

// Do when disconnected
func Disconnected() {
	Joins.All = []uint64{}
	Joins.List.Refresh()
	Ready.All = []uint64{}
	Ready.List.Refresh()
	Graveyard.All = []uint64{}
	Graveyard.List.Refresh()
	Finals.All = []uint64{}
	Finals.List.Refresh()
	Leaders.board = []records{}
	Duels.Index = make(map[uint64]entry)
	Graveyard.Index = make(map[uint64]grave)
	Inventory.ClearAll()
}

// Check if Wallet.Address is owner or ref on duel SC
func checkOwnerAndRefs() bool {
	if own, _ := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "owner"); own != nil {
		if rpc.Wallet.Address == own[0] {
			return true
		}
	}

	for i := 0; i < 10; i++ {
		if ref, _ := menu.Gnomes.GetSCIDValuesByKey(DUELSCID, "ref"+strconv.Itoa(i)); ref != nil {
			if rpc.Wallet.Address == ref[0] {
				return true
			}
		}
	}

	return false
}

// Checks if wallet has any claimable duel NFAs, looking for dst from ref transfer
func checkClaimable() (claimable []string) {
	entries := rpc.GetWalletTransfers(2450000, uint64(rpc.Wallet.Height), uint64(0xA1B2C3D4E5F67890))
	for _, e := range *entries {
		split := strings.Split(string(e.Payload), "  ")
		if len(split) > 2 && len(split[1]) == 64 {
			if menu.CheckOwner(split[1]) || rpc.TokenBalance(split[1]) != 1 {
				continue
			}

			var have bool
			for _, sc := range claimable {
				if sc == split[1] {
					have = true
					break
				}
			}

			if !have {
				claimable = append(claimable, split[1])
			}
		}
	}

	return
}

// Call ClaimOwnership on SC and confirm tx on all claimable
func claimClaimable(claimable []string, d *dreams.AppObject) {
	wait := true
	progress_label := dwidget.NewCenterLabel("")
	progress := widget.NewProgressBar()
	progress_cont := container.NewBorder(nil, progress_label, nil, nil, progress)
	progress.Min = float64(0)
	progress.Max = float64(len(claimable))
	wait_message := dialog.NewCustom("Claiming Duel Assets", "Stop", progress_cont, d.Window)
	wait_message.Resize(fyne.NewSize(610, 150))
	wait_message.SetOnClosed(func() {
		wait = false
	})
	wait_message.Show()

	for i, claim := range claimable {
		if !wait {
			break
		}

		retry := 0
		for retry < 4 {
			if !wait {
				break
			}

			progress.SetValue(float64(i))
			progress_label.SetText(fmt.Sprintf("Claiming: %s\n\nPlease wait for TX to be confirmed", claim))
			tx := rpc.ClaimNFA(claim)
			time.Sleep(time.Second)
			retry += rpc.ConfirmTxRetry(tx, "checkClaimable", 45)

			retry++

		}
	}
	progress.SetValue(progress.Value + 1)
	progress_label.SetText("Completed all claims")
	wait_message.SetDismissText("Done")
}

// func playAnimation(address string, obj *fyne.Container, reset fyne.CanvasObject) {
// 	cowboy, _ := xwidget.NewAnimatedGifFromResource(CowboysGif)
// 	cowboy.SetMinSize(fyne.NewSize(600, 336))
// 	obj.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = container.NewCenter(cowboy)
// 	obj.Objects[0].(*container.Split).Trailing.Refresh()
// 	obj.Refresh()

// 	go cowboy.Start()
// 	time.Sleep(7 * time.Second)
// 	cowboy.Stop()

// 	bottom := canvas.NewText("has won this Duel", color.Black)
// 	bottom.Alignment = fyne.TextAlignCenter

// 	winner := container.NewCenter(container.NewVBox(
// 		canvas.NewText(address, color.Black),
// 		bottom))

// 	obj.Objects[0].(*container.Split).Trailing.(*fyne.Container).Add(winner)
// 	time.Sleep(7 * time.Second)

// 	obj.Objects[0].(*container.Split).Trailing.(*fyne.Container).Remove(winner)
// 	obj.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = reset
// 	obj.Objects[0].(*container.Split).Trailing.Refresh()
// }
