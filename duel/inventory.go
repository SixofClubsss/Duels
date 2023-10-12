package duel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	dreams "github.com/dReam-dApps/dReams"
	"github.com/dReam-dApps/dReams/bundle"
	"github.com/dReam-dApps/dReams/menu"
	"github.com/dReam-dApps/dReams/rpc"
)

type asset struct {
	rank uint64
	img  []byte
}

type assetRank struct {
	Rank int `json:"r"`
}

type inventory struct {
	Character  dreams.AssetSelect
	Item1      dreams.AssetSelect
	Item2      dreams.AssetSelect
	characters map[string]asset
	items      map[string]asset
	sync.RWMutex
}

var asset_ranks = []string{" {R1}", " {R2}", " {R3}", " {R4}", " {R5}"}
var Inventory inventory

// Creates a 100x100 icon with frame
//   - Pass icon image as []byte
func iconLarge(icon []byte, name string) fyne.CanvasObject {
	frame := canvas.NewImageFromResource(bundle.ResourceAvatarFramePng)
	frame.SetMinSize(fyne.NewSize(100, 100))

	if icon == nil {
		icon = resourceUnknownIconPng.StaticContent
	}

	canv := canvas.NewImageFromReader(bytes.NewReader(icon), name)
	if canv == nil {
		return container.NewMax(frame)
	}

	canv.SetMinSize(fyne.NewSize(90, 90))
	border := container.NewBorder(layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer(), canv)

	return container.NewMax(border, frame)
}

// Creates a 60x60 icon with frame
//   - Pass icon image as []byte
//   - Pass died as true to X out icon
func iconSmall(icon []byte, name string, died bool) fyne.CanvasObject {
	frame := canvas.NewImageFromResource(bundle.ResourceAvatarFramePng)
	frame.SetMinSize(fyne.NewSize(60, 60))

	if icon == nil {
		icon = resourceUnknownIconPng.StaticContent
	}

	canv := canvas.NewImageFromReader(bytes.NewReader(icon), name)
	if canv == nil {
		return container.NewMax(frame)
	}

	canv.SetMinSize(fyne.NewSize(55, 55))
	border := container.NewBorder(layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer(), canv)

	max := container.NewMax(border)
	if died {
		x := canvas.NewImageFromResource(resourceDiedPng)
		x.SetMinSize(fyne.NewSize(60, 60))
		max.Add(x)
	}

	max.Add(frame)

	return max
}

// Remove {R#} rank from asset display string
func removeRank(str string) (newStr string) {
	newStr = str
	for _, ending := range asset_ranks {
		if strings.HasSuffix(str, ending) {
			newStr = strings.TrimSuffix(str, ending)
		}
	}

	return
}

// Check if SCID has a valid duel rank
func validateAssetRank(scid string) uint64 {
	if desc, _ := menu.Gnomes.GetSCIDValuesByKey(scid, "descrHdr"); desc != nil {
		var rank assetRank
		split := strings.Split(desc[0], ";;")

		if len(split) < 2 {
			return 1
		}

		if err := json.Unmarshal([]byte(rpc.HexToString(split[1])), &rank); err != nil {
			logger.Errorln("[validateAsset]", err)
			return 1
		}

		return uint64(rank.Rank)
	}

	return 0
}

// Add duel character asset to inventory and download icon image
func (inv *inventory) AddCharToInventory(name string) {
	scid := menu.Assets.Asset_map[name]
	img, err := downloadBytes(scid)
	if err != nil {
		inv.characters[name] = asset{
			rank: 0,
			img:  resourceUnknownIconPng.StaticContent,
		}
		return
	}

	inv.Lock()
	inv.characters[name] = asset{
		rank: validateAssetRank(scid),
		img:  img,
	}
	inv.Unlock()
}

// Add duel item asset to inventory and download icon image
func (inv *inventory) AddItemToInventory(name string) {
	scid := menu.Assets.Asset_map[name]
	img, err := downloadBytes(scid)
	if err != nil {
		inv.items[name] = asset{
			rank: 0,
			img:  resourceUnknownIconPng.StaticContent,
		}
		return
	}

	inv.Lock()
	inv.items[name] = asset{
		rank: validateAssetRank(scid),
		img:  img,
	}
	inv.Unlock()
}

// Sort all inventory select options
func (inv *inventory) SortAll() {
	inv.Character.Sort()
	inv.Item1.Sort()
	inv.Item2.Sort()
}

// Clear all inventory select options
func (inv *inventory) ClearAll() {
	inv.Character.Select.Selected = ""
	inv.Character.ClearAll()
	inv.Item1.Select.Selected = ""
	inv.Item1.ClearAll()
	inv.Item2.Select.Selected = ""
	inv.Item2.ClearAll()
}

// Returns the rank of all currently selected items
func (inv *inventory) findRank() (rank uint64) {
	inv.RLock()
	defer inv.RUnlock()
	if inv.Character.Select.Selected != "" {
		rank += inv.characters[removeRank(inv.Character.Select.Selected)].rank
	}

	if inv.Item1.Select.Selected != "" {
		rank += inv.items[removeRank(inv.Item1.Select.Selected)].rank
	}

	if inv.Item2.Select.Selected != "" {
		rank += inv.items[removeRank(inv.Item2.Select.Selected)].rank
	}

	return
}

// Add duel assets to inventory, adding rank to name for display string
// Implemented characters or items without a rank are hard coded rank 1
func AddItemsToInventory(scid, header, owner, collection string) {
	if rpc.TokenBalance(scid) != 1 {
		logger.Debugf("[AddItemsToInventory] %s token not in wallet\n", scid)
		return
	}

	if desc, _ := menu.Gnomes.GetSCIDValuesByKey(scid, "descrHdr"); desc != nil {
		var rank assetRank
		splitDesc := strings.Split(desc[0], ";;")
		if len(splitDesc) > 1 {
			// Ranked assets
			switch collection {
			case "Dero Desperados":
				if err := json.Unmarshal([]byte(rpc.HexToString(splitDesc[1])), &rank); err != nil {
					logger.Errorln("[AddItemsToInventory]", err)
					return
				}

				Inventory.Character.Add(fmt.Sprintf("%s {R%d}", header, rank.Rank), owner)
				go Inventory.AddCharToInventory(header)
			case "Desperado Guns":
				if err := json.Unmarshal([]byte(rpc.HexToString(splitDesc[1])), &rank); err != nil {
					logger.Errorln("[AddItemsToInventory]", err)
					return
				}

				Inventory.Item1.Add(fmt.Sprintf("%s {R%d}", header, rank.Rank), owner)
				Inventory.Item2.Add(fmt.Sprintf("%s {R%d}", header, rank.Rank), owner)
				go Inventory.AddItemToInventory(header)
			case "TestChars":
				if err := json.Unmarshal([]byte(rpc.HexToString(splitDesc[1])), &rank); err != nil {
					logger.Errorln("[AddItemsToInventory]", err)
					return
				}

				Inventory.Character.Add(fmt.Sprintf("%s {R%d}", header, rank.Rank), owner)
				go Inventory.AddCharToInventory(header)
			case "TestItems":
				if err := json.Unmarshal([]byte(rpc.HexToString(splitDesc[1])), &rank); err != nil {
					logger.Errorln("[AddItemsToInventory]", err)
					return
				}

				Inventory.Item1.Add(fmt.Sprintf("%s {R%d}", header, rank.Rank), owner)
				Inventory.Item2.Add(fmt.Sprintf("%s {R%d}", header, rank.Rank), owner)
				go Inventory.AddItemToInventory(header)
			}
		} else {
			rank := 1
			switch collection {
			case "High Strangeness":
				Inventory.Character.Add(fmt.Sprintf("%s {R%d}", header, rank), owner)
				go Inventory.AddCharToInventory(header)
			}
		}
	}
}
