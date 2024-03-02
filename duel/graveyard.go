package duel

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"github.com/dReam-dApps/dReams/dwidget"
	"github.com/dReam-dApps/dReams/rpc"
)

type graveMap struct {
	Index map[uint64]grave
	dwidget.Lists
	sync.RWMutex
}

type grave struct {
	Num   string `json:"num"`
	Char  string
	Token string `json:"token"`
	Amt   uint64
	Time  int64
	Icon  []byte
}

var Graveyard graveMap

// Get asset name of grave
func (grave grave) assetName() (asset_name string) {
	asset_name = "DERO"
	if name := rpc.GetAssetNameBySCID(grave.Token); name != "" {
		asset_name = name
	}

	return
}

// Get icon image for graveyard character
//   - size 0 for small icon and 1 for large
func (grave grave) IconImage(size int) fyne.CanvasObject {
	if size == 0 {
		return iconSmall(grave.Icon, grave.Char, false)
	}

	return iconLarge(grave.Icon, grave.Char)
}

// Find graveyard discount if applicable, discounted 20% per week
func (grave grave) findDiscount() (discount uint64) {
	perc := uint64(100)
	duration := time.Duration(time.Now().Unix()-grave.Time) * time.Second

	switch {
	case duration >= 28*24*time.Hour:
		perc = 20
	case duration >= 21*24*time.Hour:
		perc = 40
	case duration >= 14*24*time.Hour:
		perc = 60
	case duration >= 7*24*time.Hour:
		perc = 80
	}

	return perc * grave.Amt / 100
}

// Gets graveyard and leader board data
func GetGraveyard() {
	if gnomon.IsReady() {
		if info := gnomon.GetAllSCIDVariableDetails(DUELSCID); info != nil {
			Graveyard.Lock()
			defer Graveyard.Unlock()
			for _, h := range info {
				updateSyncProgress(sync_prog)
				if str, ok := h.Key.(string); ok {
					split := strings.Split(str, "_")
					switch split[0] {
					case "ret":
						u := rpc.StringToUint64(split[1])
						if Graveyard.Index[u].Char == "" {

							if _, time := gnomon.GetSCIDValuesByKey(DUELSCID, "time_"+split[1]+"_"+split[2]); time != nil {
								img, err := downloadBytes(split[2])
								if err != nil {
									logger.Errorln("[GetGraveyard]", h.Key, err)
									continue
								}

								token, _ := gnomon.GetSCIDValuesByKey(DUELSCID, "tkn_"+split[1])
								if token == nil {
									token = append(token, "")
								}

								Graveyard.All = append(Graveyard.All, u)

								Graveyard.Index[u] = grave{
									Num:   strconv.FormatUint(u, 10),
									Char:  split[2],
									Token: token[0],
									Amt:   rpc.Uint64Type(h.Value),
									Time:  int64(time[0]),
									Icon:  img,
								}

								Graveyard.SortIndex(false)
							}
						}

					case "w":
						addr := rpc.DeroAddressFromKey(split[1])
						if len(addr) == 66 {
							var i int
							var have bool
							for c, r := range Leaders.board {
								if r.address == addr {
									have = true
									i = c
									break
								}
							}

							_, losses := gnomon.GetSCIDValuesByKey(DUELSCID, "l_"+split[1])
							if losses == nil {
								losses = append(losses, 0)
							}

							if have {
								Leaders.board[i] = records{
									address: addr,
									value: record{
										Win:  rpc.Uint64Type(h.Value),
										Loss: losses[0]},
								}
								continue
							}

							Leaders.board = append(Leaders.board, records{
								address: addr,
								value: record{
									Win:  rpc.Uint64Type(h.Value),
									Loss: losses[0]},
							})
						}

					default:
						continue
					}
				}
			}

			// Remove revived from Graveyard
			for u, gr := range Graveyard.Index {
				str := strconv.Itoa(int(u))
				if _, re := gnomon.GetSCIDValuesByKey(DUELSCID, "ret_"+str+"_"+gr.Char); re == nil {
					Graveyard.RemoveIndex(u)
					Graveyard.Index[u] = grave{}
				}
			}

			// Sort leader board
			sort.Slice(Leaders.board, func(i, j int) bool {
				return Leaders.board[i].value.Win > Leaders.board[j].value.Win
			})
		}
		Leaders.list.Refresh()
		Graveyard.List.Refresh()
	}
}
