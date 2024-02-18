package duel

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	dreams "github.com/dReam-dApps/dReams"
	"github.com/dReam-dApps/dReams/bundle"
	"github.com/dReam-dApps/dReams/dwidget"
	"github.com/dReam-dApps/dReams/gnomes"
	"github.com/dReam-dApps/dReams/menu"
	"github.com/dReam-dApps/dReams/rpc"
)

type searches struct {
	searching bool
	results   []uint64
}
type searching struct {
	joins   searches
	ready   searches
	graves  searches
	results searches
}

var search searching
var sync_prog *widget.ProgressBar

// Layout all duel items
func LayoutAllItems(asset_map map[string]string, d *dreams.AppObject) fyne.CanvasObject {
	selected_join := uint64(0)
	selected_duel := uint64(0)
	selected_grave := uint64(0)
	var resetToTabs, updateDMLabel func()
	var starting_duel, accepting_duel, loaded bool
	var max *fyne.Container
	var tabs *container.AppTabs

	// Character and item selection objects
	silhouette := canvas.NewImageFromResource(resourceDuelistSilPng)
	silhouette.SetMinSize(fyne.NewSize(320, 320))

	total_rank_label := dwidget.NewCanvasText("", 18, fyne.TextAlignCenter)

	start_duel := widget.NewButton("Start Duel", nil)
	start_duel.Importance = widget.HighImportance
	start_duel.Hide()

	character := canvas.NewImageFromResource(bundle.ResourceFramePng)
	character.SetMinSize(fyne.NewSize(100, 100))

	Inventory.Character.Select = widget.NewSelect([]string{}, nil)
	Inventory.Character.Select.PlaceHolder = "Character:"

	char_clear := widget.NewButtonWithIcon("", dreams.FyneIcon("viewRefresh"), nil)
	char_clear.Importance = widget.LowImportance
	char_clear.OnTapped = func() {
		Inventory.Character.Select.ClearSelected()
	}

	character_cont := container.NewBorder(
		container.NewBorder(nil, nil, char_clear, nil, Inventory.Character.Select),
		nil,
		nil,
		nil,
		container.NewCenter(character))

	var item1_cont, item2_cont *fyne.Container

	item1 := canvas.NewImageFromResource(bundle.ResourceFramePng)
	item1.SetMinSize(fyne.NewSize(100, 100))

	item2 := canvas.NewImageFromResource(bundle.ResourceFramePng)
	item2.SetMinSize(fyne.NewSize(100, 100))

	Inventory.Character.Select.OnChanged = func(s string) {
		go func() {
			if s == "" {
				Inventory.Item1.Select.Disable()
				Inventory.Item2.Select.Disable()
				Inventory.Item2.Select.ClearSelected()
				Inventory.Item1.Select.ClearSelected()

				item1_cont.Objects[1].(*fyne.Container).Objects[0] = item1
				item2_cont.Objects[1].(*fyne.Container).Objects[0] = item2
				start_duel.Hide()
				character_cont.Objects[0].(*fyne.Container).Objects[0] = character
				if starting_duel {
					resetToTabs()
					info := dialog.NewInformation("Start Duel", "Choose a character to duel with", d.Window)
					info.SetOnClosed(func() {
						Inventory.Character.Select.FocusLost()
					})
					info.Show()
					Inventory.Character.Select.FocusGained()

				}
				total_rank_label.Text = ""
				total_rank_label.Refresh()

				return
			}

			total_rank_label.Text = fmt.Sprintf("You're Rank: (R%d)", Inventory.findRank())
			total_rank_label.Refresh()

			if selected_join == 0 {
				start_duel.Show()
			}
			Inventory.Item1.Select.Enable()
			Inventory.RLock()
			character_cont.Objects[0].(*fyne.Container).Objects[0] = iconLarge(Inventory.characters[removeRank(s)].img, s)
			Inventory.RUnlock()
		}()
	}

	select_spacer := canvas.NewRectangle(color.RGBA{0, 0, 0, 0})
	select_spacer.SetMinSize(fyne.NewSize(120, 0))

	Inventory.Item1.Select = widget.NewSelect([]string{}, nil)
	Inventory.Item1.Select.PlaceHolder = "Item 1:"

	item1_clear := widget.NewButtonWithIcon("", dreams.FyneIcon("viewRefresh"), nil)
	item1_clear.Importance = widget.LowImportance
	item1_clear.OnTapped = func() {
		Inventory.Item1.Select.ClearSelected()
	}

	item1_cont = container.NewVBox(container.NewBorder(nil, nil, item1_clear, nil, container.NewStack(select_spacer, Inventory.Item1.Select)), container.NewCenter(item1))

	Inventory.Item1.Select.OnChanged = func(s string) {
		updateDMLabel()
		if s == "" {
			Inventory.Item2.Select.Disable()
			Inventory.Item2.Select.Selected = ""
			Inventory.Item2.Select.Options = Inventory.Item1.Select.Options
			Inventory.Item2.Select.Refresh()
			item1_cont.Objects[1].(*fyne.Container).Objects[0] = item1
			if Inventory.Character.Select.Selected == "" {
				Inventory.Item1.Select.Disable()
				total_rank_label.Text = ""
				total_rank_label.Refresh()
			} else {
				Inventory.Item1.Select.Enable()
			}
			return
		}

		total_rank_label.Text = fmt.Sprintf("You're Rank: (R%d)", Inventory.findRank())
		total_rank_label.Refresh()

		Inventory.RLock()
		item1_cont.Objects[1].(*fyne.Container).Objects[0] = iconLarge(Inventory.items[removeRank(s)].img, s)
		Inventory.RUnlock()
		Inventory.Item2.Select.Enable()
	}
	Inventory.Item1.Select.Disable()

	Inventory.Item2.Select = widget.NewSelect([]string{}, nil)
	Inventory.Item2.Select.PlaceHolder = "Item 2:"

	item2_clear := widget.NewButtonWithIcon("", dreams.FyneIcon("viewRefresh"), nil)
	item2_clear.Importance = widget.LowImportance
	item2_clear.OnTapped = func() {
		Inventory.Item2.Select.ClearSelected()
	}

	item2_cont = container.NewVBox(container.NewBorder(nil, nil, item2_clear, nil, container.NewStack(select_spacer, Inventory.Item2.Select)), container.NewCenter(item2))

	Inventory.Item2.Select.OnChanged = func(s string) {
		updateDMLabel()
		if s == "" {
			if Inventory.Item1.Select.Selected != "" {
				if rpc.IsReady() {
					Inventory.Item1.Select.Enable()
				}
			} else {
				if Inventory.Character.Select.Selected == "" {
					total_rank_label.Text = ""
					total_rank_label.Refresh()
				}
			}
			item2_cont.Objects[1].(*fyne.Container).Objects[0] = item2
			return
		}

		total_rank_label.Text = fmt.Sprintf("You're Rank: (R%d)", Inventory.findRank())
		total_rank_label.Refresh()

		Inventory.RLock()
		item2_cont.Objects[1].(*fyne.Container).Objects[0] = iconLarge(Inventory.items[removeRank(s)].img, s)
		Inventory.RUnlock()

		Inventory.Item1.Select.Disable()
	}
	Inventory.Item2.Select.Disable()

	equip_alpha := canvas.NewRectangle(color.Transparent)
	equip_alpha.SetMinSize(fyne.NewSize(450, 600))

	equip_spacer := canvas.NewRectangle(color.Transparent)
	equip_spacer.SetMinSize(fyne.NewSize(150, 300))

	equip_box := container.NewCenter(equip_alpha,
		container.NewStack(
			silhouette,
			container.NewBorder(
				container.NewStack(character_cont),
				container.NewBorder(nil, nil, nil, nil, container.NewStack()),
				container.NewStack(item1_cont),
				container.NewStack(item2_cont),
				equip_spacer)))

	sync_label := dwidget.NewCenterLabel("Connect to Daemon and Wallet to sync")

	sync_prog = widget.NewProgressBar()
	sync_prog.Min = 0
	sync_prog.Max = 50
	sync_prog.TextFormatter = func() string {
		return ""
	}

	sync_spacer := canvas.NewRectangle(color.Transparent)
	sync_spacer.SetMinSize(fyne.NewSize(485, 0))
	sync_img := canvas.NewImageFromResource(ResourceDuelCirclePng)
	sync_img.SetMinSize(fyne.NewSize(200, 200))
	sync_cont := container.NewStack(container.NewCenter(container.NewBorder(sync_spacer, sync_label, nil, nil, container.NewCenter(sync_img))), sync_prog)

	options_select := widget.NewSelect([]string{"Rescan", "Claim All", "Clear Cache"}, nil)
	options_select.PlaceHolder = "Options:"

	equip_cont := container.NewBorder(
		container.NewCenter(container.NewVBox(canvas.NewLine(bundle.TextColor), dwidget.NewCanvasText("Your Inventory", 18, fyne.TextAlignCenter), canvas.NewLine(bundle.TextColor))),
		container.NewBorder(nil, nil, container.NewStack(dwidget.NewSpacer(90, 36), start_duel), options_select, container.NewCenter(container.NewVBox(total_rank_label, dwidget.NewLine(150, 0, bundle.TextColor)))),
		nil,
		nil,
		equip_box)

	options_select.OnChanged = func(s string) {
		switch s {
		case "Rescan":
			if rpc.IsReady() {
				dialog.NewConfirm("Rescan", "Would you like to rescan wallet for Duel assets?", func(b bool) {
					if b {
						total_rank_label.Text = ""
						sync_label.SetText("Scanning Assets...")
						max.Objects[0].(*container.Split).Leading.(*fyne.Container).Objects[0] = container.NewStack(sync_cont)
						Inventory.ClearAll()
						character_cont.Objects[0].(*fyne.Container).Objects[0] = character
						item1_cont.Objects[1].(*fyne.Container).Objects[0] = item1
						item2_cont.Objects[1].(*fyne.Container).Objects[0] = item2
						checkNFAs("Duels", false, true, nil)
						sync_prog.Max = 1
						sync_prog.SetValue(0)
						max.Objects[0].(*container.Split).Leading.(*fyne.Container).Objects[0] = container.NewStack(equip_cont)
					}
				}, d.Window).Show()
			} else {
				dialog.NewInformation("Rescan", "You are not connected to daemon or wallet", d.Window).Show()
			}
		case "Claim All":
			if rpc.IsReady() {
				claimable := menu.CheckClaimable()
				l := len(claimable)
				if l > 0 {
					dialog.NewConfirm("Claim All", fmt.Sprintf("Claim your %d available assets?", l), func(b bool) {
						if b {
							go menu.ClaimClaimable("Claiming Duel Assets", claimable, d)
						}
					}, d.Window).Show()
				} else {
					dialog.NewInformation("Claim All", "You have no claimable assets", d.Window).Show()
				}
			} else {
				dialog.NewInformation("Claim All", "You are not connected to daemon or wallet", d.Window).Show()
			}
		case "Clear Cache":
			if gnomon.DBStorageType() == "boltdb" {
				dialog.NewConfirm("Clear Image Cache", "Would you like to clear your stored image cache?", func(b bool) {
					if b {
						gnomes.DeleteStorage("DUELBUCKET", "DUELS")
					}
				}, d.Window).Show()
			} else {
				dialog.NewInformation("Clear Cache", "You have no stored image cache", d.Window).Show()
			}
		default:
			// nothing
		}
		options_select.Selected = ""
	}

	// Opponent box
	opponent_sil := canvas.NewImageFromResource(resourceOpponentSilPng)
	opponent_sil.SetMinSize(fyne.NewSize(480, 480))

	opponent_character := canvas.NewImageFromResource(bundle.ResourceFramePng)
	opponent_character.SetMinSize(fyne.NewSize(100, 100))
	opponent_character_cont := container.NewCenter(opponent_character)

	opponent_item1 := canvas.NewImageFromResource(bundle.ResourceFramePng)
	opponent_item1.SetMinSize(fyne.NewSize(100, 100))

	opponent_item1_cont := container.NewVBox(container.NewCenter(opponent_item1))

	opponent_item2 := canvas.NewImageFromResource(bundle.ResourceFramePng)
	opponent_item2.SetMinSize(fyne.NewSize(100, 100))
	opponent_item2_cont := container.NewVBox(container.NewCenter(opponent_item2))

	opponent_alpha := canvas.NewRectangle(color.Transparent)
	opponent_alpha.SetMinSize(fyne.NewSize(450, 500))

	opponent_label := widget.NewLabel("")
	opponent_label.Alignment = fyne.TextAlignCenter

	opponent_spacer := canvas.NewRectangle(color.Transparent)
	opponent_spacer.SetMinSize(fyne.NewSize(150, 300))

	opponent_equip_box := container.NewCenter(
		opponent_alpha,
		container.NewStack(
			opponent_sil,
			container.NewBorder(opponent_character_cont, layout.NewSpacer(), opponent_item1_cont, opponent_item2_cont, opponent_spacer)))

	accept_duel := widget.NewButton("Accept Duel", nil)
	accept_duel.Importance = widget.HighImportance

	resetToTabs = func() {
		selected_join = 0
		selected_duel = 0
		selected_grave = 0
		starting_duel = false
		accepting_duel = false
		max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = tabs
		Joins.List.UnselectAll()
		Ready.List.UnselectAll()
		Graveyard.List.UnselectAll()
		Finals.List.UnselectAll()
		opponent_label.SetText("")
		if rpc.IsReady() && Inventory.Character.Select.Selected != "" {
			start_duel.Show()
		}
		char_clear.Enable()
		item1_clear.Enable()
		item2_clear.Enable()

		Inventory.Character.Select.Enable()
		if Inventory.Character.Select.Selected != "" {
			Inventory.Item1.Select.Enable()
		}

		if Inventory.Item1.Select.Selected != "" {
			Inventory.Item2.Select.Enable()
		}
	}

	back_button := widget.NewButton("Back", func() {
		resetToTabs()
	})

	opponent_equip_cont := container.NewBorder(nil, container.NewCenter(container.NewAdaptiveGrid(2, accept_duel, back_button)), nil, nil, opponent_equip_box)

	// List of current joinable duels
	Joins.List = widget.NewList(
		func() int {
			return len(Joins.All)
		},
		func() fyne.CanvasObject {
			return container.NewBorder(
				dwidget.NewCenterLabel(""),
				container.NewVBox(
					dwidget.NewCenterLabel(""),
					dwidget.NewCenterLabel("")),
				nil,
				nil,
				container.NewCenter(container.NewHBox(iconSmall(nil, "", false), layout.NewSpacer(), layout.NewSpacer())))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			go func() {
				var id uint64
				if search.joins.searching {
					if search.joins.results == nil {
						return
					}
					id = search.joins.results[i]
				} else {
					if i+1 > len(Joins.All) {
						return
					}
					id = Joins.All[i]
				}

				Duels.RLock()
				defer Duels.RUnlock()

				header := fmt.Sprintf("Duel #%s   Rank: (R%d)   Amount: (%s %s)   Items: (%d)   Death Match: (%s)   Hardcore: (%s)", Duels.Index[id].Num, Duels.Index[id].getDuelistRank(), rpc.FromAtomic(Duels.Index[id].Amt, 5), Duels.Index[id].assetName(), Duels.Index[id].Items, Duels.Index[id].DM, Duels.Index[id].Rule)

				if Duels.Index[id].Num != "" && !Duels.Index[id].Complete && o.(*fyne.Container).Objects[1].(*widget.Label).Text != header {
					o.(*fyne.Container).Objects[2].(*fyne.Container).Objects[1].(*widget.Label).SetText(fmt.Sprintf("Duelist: %s %s", Duels.Index[id].Duelist.Address, Leaders.getRecordByAddress(Duels.Index[id].Duelist.Address)))
					o.(*fyne.Container).Objects[2].(*fyne.Container).Objects[0].(*widget.Label).SetText(Duels.Index[id].Duelist.getRankString())
					o.(*fyne.Container).Objects[1].(*widget.Label).SetText(header)

					if Duels.Index[id].Items > 1 {
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 0)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1] = Duels.Index[id].Duelist.IconImage(0, 1)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2] = Duels.Index[id].Duelist.IconImage(0, 2)
						o.Refresh()
						return
					}

					if Duels.Index[id].Items > 0 {
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 0)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1] = Duels.Index[id].Duelist.IconImage(0, 1)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2] = layout.NewSpacer()
						o.Refresh()
						return
					}

					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 0)
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1] = layout.NewSpacer()
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2] = layout.NewSpacer()
					o.Refresh()
				}
			}()
		})

	// Clicking on a join will bring up join confirmation
	Joins.List.OnSelected = func(id widget.ListItemID) {
		if !loaded {
			return
		}
		go func() {
			if search.joins.searching {
				if search.joins.results == nil {
					return
				}
				selected_join = search.joins.results[id]
			} else {
				if id+1 > len(Joins.All) {
					return
				}
				selected_join = Joins.All[id]
			}

			Duels.RLock()
			defer Duels.RUnlock()

			validated := Duels.Index[selected_join].validateCollection(false)

			if rpc.Wallet.Address == Duels.Index[selected_join].Duelist.Address || (checkOwnerAndRefs() && !validated) {
				prefix := "W"
				if !validated {
					prefix = "Duelist not valid, w"
				}
				dialog.NewConfirm("Cancel Duel", fmt.Sprintf("%sould you like to cancel this Duel?", prefix), func(b bool) {
					if b {
						if n := strconv.FormatUint(selected_join, 10); n != "" {
							tx := Refund(n)
							go menu.ShowTxDialog("Cancel Duel", "Duels", tx, 3*time.Second, d.Window)

							resetToTabs()
						}
					}
					Joins.List.UnselectAll()
					start_duel.Show()

				}, d.Window).Show()

				return
			}

			if !validated {
				dialog.NewInformation("Invalid Collection", "This assets collection can not be validated", d.Window).Show()
				Joins.List.UnselectAll()
				return
			}

			items := Duels.Index[selected_join].Items
			stamp := Duels.Index[selected_join].Stamp
			now := time.Now().Unix()

			if now < stamp {
				dialog.NewInformation("Join Buffer", fmt.Sprintf("This Duel will be joinable in %d seconds", stamp-now), d.Window).Show()
				resetToTabs()
				return
			}

			switch items {
			case 0:
				if Inventory.Character.Select.Selected == "" {
					info := dialog.NewInformation("No Character", "Select a character", d.Window)
					info.SetOnClosed(func() {
						Inventory.Character.Select.FocusLost()
					})
					info.Show()
					Inventory.Character.Select.FocusGained()
					resetToTabs()
					return
				}

				if Inventory.Item1.Select.Selected != "" || Inventory.Item2.Select.Selected != "" {
					info := dialog.NewInformation("No items", "No item can be used for this duel", d.Window)
					info.SetOnClosed(func() {
						item1_clear.FocusLost()
						item2_clear.FocusLost()
					})
					info.Show()
					if Inventory.Item1.Select.Selected != "" {
						item1_clear.FocusGained()
					}
					if Inventory.Item2.Select.Selected != "" {
						item2_clear.FocusGained()
					}
					resetToTabs()
					return
				}
			case 1:
				if Inventory.Character.Select.Selected == "" {
					info := dialog.NewInformation("No Character", "Select a character", d.Window)
					info.SetOnClosed(func() {
						Inventory.Character.Select.FocusLost()
					})
					info.Show()
					Inventory.Character.Select.FocusGained()
					resetToTabs()
					return
				}

				if Inventory.Item1.Select.Selected == "" {
					info := dialog.NewInformation("No Item", "A item is required for this duel", d.Window)
					info.SetOnClosed(func() {
						Inventory.Item1.Select.FocusLost()
					})
					info.Show()
					Inventory.Item1.Select.FocusGained()
					resetToTabs()
					return
				}

				if Inventory.Item2.Select.Selected != "" {
					info := dialog.NewInformation("One Item Only", "Only one item can be used for this duel", d.Window)
					info.SetOnClosed(func() {
						item2_clear.FocusLost()
					})
					info.Show()
					item2_clear.FocusGained()
					resetToTabs()
					return
				}

			case 2:
				if Inventory.Character.Select.Selected == "" {
					info := dialog.NewInformation("No Character", "Select a character", d.Window)
					info.SetOnClosed(func() {
						Inventory.Character.Select.FocusLost()
					})
					info.Show()
					Inventory.Character.Select.FocusGained()
					resetToTabs()
					return
				}

				if Inventory.Item1.Select.Selected == "" || Inventory.Item2.Select.Selected == "" {
					info := dialog.NewInformation("No Items", "Two items are required for this duel", d.Window)
					info.SetOnClosed(func() {
						Inventory.Item1.Select.FocusLost()
						Inventory.Item2.Select.FocusLost()
					})
					info.Show()
					Inventory.Item1.Select.FocusGained()
					Inventory.Item2.Select.FocusGained()
					resetToTabs()
					return
				}

				if Inventory.Item1.Select.SelectedIndex() >= 0 && Inventory.Item1.Select.Selected == Inventory.Item2.Select.Selected {
					info := dialog.NewInformation("Same Item", "You can't use the same item twice", d.Window)
					info.SetOnClosed(func() {
						Inventory.Item2.Select.FocusLost()
					})
					info.Show()
					Inventory.Item2.Select.FocusGained()
					resetToTabs()
					return
				}

			default:
				resetToTabs()
				return
			}

			char_clear.Disable()
			item1_clear.Disable()
			item2_clear.Disable()

			Inventory.Character.Select.Disable()
			Inventory.Item1.Select.Disable()
			Inventory.Item2.Select.Disable()

			start_duel.Hide()

			char_cont := container.NewCenter(Duels.Index[selected_join].Duelist.IconImage(1, 0))

			opponent_item1_cont = container.NewCenter()
			opponent_item2_cont = container.NewCenter()

			if items > 0 {
				icon := Duels.Index[selected_join].Duelist.IconImage(1, 1)
				opponent_item1_cont = container.NewCenter(icon)
			}

			if items > 1 {
				icon := Duels.Index[selected_join].Duelist.IconImage(1, 2)
				opponent_item2_cont = container.NewCenter(icon)
			}

			r2 := Inventory.findRank()
			perc, r1, diff := Duels.Index[selected_join].diffOdds(r2)

			opponent_equip_box = container.NewCenter(
				opponent_alpha,
				container.NewStack(
					opponent_sil,
					container.NewBorder(
						container.NewVBox(dwidget.NewSpacer(0, 35), char_cont),
						container.NewCenter(widget.NewLabel(fmt.Sprintf("Opponent Rank: (R%d)   Death match: (%s)   Hardcore: (%s)", r1, Duels.Index[selected_join].DM, Duels.Index[selected_join].Rule))),
						opponent_item1_cont,
						opponent_item2_cont,
						opponent_spacer)))

			amt := Duels.Index[selected_join].Amt
			asset_name := Duels.Index[selected_join].assetName()

			label_text := "Evenly ranked, "
			if Duels.Index[selected_join].Rule == "Yes" {
				perc = uint64(100)
				label_text = "Hardcore mode, "
			} else {
				if r2 > r1 {
					if diff == 1 {
						label_text = fmt.Sprintf("You are %d rank higher, ", diff)
					} else {
						label_text = fmt.Sprintf("You are %d ranks higher, ", diff)
					}
				}

				if r1 > r2 {
					if i := r1 - r2; i == 1 {
						label_text = fmt.Sprintf("You are %d rank below, ", i)
					} else {
						label_text = fmt.Sprintf("You are %d ranks below, ", i)
					}
				}
			}

			label_text = label_text + fmt.Sprintf("your payout for this duel will be %s %s", rpc.FromAtomic(perc*(95*(amt*2)/100)/100, 5), asset_name)

			header := "Duel"
			if Duels.Index[selected_join].DM == "Yes" {
				header = "Death Match"
				revival := fmt.Sprintf("%s %s", rpc.FromAtomic((amt*2)*(items+1), 5), asset_name)
				opponent_label.SetText(label_text + "\n\nCaution, this is a death match. If you loose this duel your items will be sent to the winner\n\nThe losers character will go to the graveyard, characters can be revived from the graveyard by anyone\n\n The revival fee for this duel will initially be " + revival + " and be discounted 20% per week")
			} else {
				opponent_label.SetText(label_text)
			}

			opponent_equip_cont = container.NewBorder(
				container.NewCenter(container.NewVBox(dwidget.NewCanvasText(fmt.Sprintf("Accept %s for %s %s", header, rpc.FromAtomic(amt, 5), asset_name), 18, fyne.TextAlignCenter), canvas.NewLine(bundle.TextColor))),
				container.NewVBox(opponent_label, container.NewCenter(container.NewAdaptiveGrid(2, accept_duel, back_button))),
				nil,
				nil,
				opponent_equip_box)

			max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = container.NewStack(bundle.NewAlpha180(), opponent_equip_cont)
			Joins.List.UnselectAll()
		}()
	}

	// List of duels ready for ref
	Ready.List = widget.NewList(
		func() int {
			return len(Ready.All)
		},
		func() fyne.CanvasObject {
			return container.NewBorder(
				dwidget.NewCenterLabel(""),
				dwidget.NewCenterLabel(""),
				nil,
				nil,
				container.NewCenter( // 0
					container.NewHBox( // 0-0
						container.NewVBox( // 0-0-0
							container.NewHBox( // 0-0-0-0
								dwidget.NewCenterLabel(""),                    // 0-0-0-0-0
								container.NewStack(iconSmall(nil, "", false)), // 0-0-0-0-1
								container.NewStack(layout.NewSpacer()),        // 0-0-0-0-2
								container.NewStack(layout.NewSpacer())),       // 0-0-0-0-3
							dwidget.NewTrailingLabel("")), // 0-0-0-1

						widget.NewSeparator(),                        // 0-0-1
						canvas.NewText("   VS   ", bundle.TextColor), // 0-0-2
						widget.NewSeparator(),                        // 0-0-3

						container.NewVBox( // 0-0-4
							container.NewHBox( // 0-0-4-0
								container.NewStack(layout.NewSpacer()), // 0-0-4-0-0
								container.NewStack(layout.NewSpacer()), // 0-0-4-0-1
								container.NewStack(layout.NewSpacer()), // 0-0-4-0-2
								dwidget.NewCenterLabel("")),            // 0-0-4-0-3
							widget.NewLabel(""))))) // 0-0-4-1
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			go func() {
				var id uint64
				if search.ready.searching {
					if search.ready.results == nil {
						return
					}
					id = search.ready.results[i]
				} else {
					if i+1 > len(Ready.All) {
						return
					}
					id = Ready.All[i]
				}

				Duels.RLock()
				defer Duels.RUnlock()

				o.(*fyne.Container).Objects[2].(*widget.Label).SetText(fmt.Sprintf("Ready for: %v", Duels.Index[id].readySince()))

				header := fmt.Sprintf("Duel #%s   Pot: (%s %s)   Items: (%d)   Death Match: (%s)", Duels.Index[id].Num, rpc.FromAtomic(Duels.Index[id].Amt*2, 5), Duels.Index[id].assetName(), Duels.Index[id].Items, Duels.Index[id].DM)
				if Duels.Index[id].Opponent.Char != "" && o.(*fyne.Container).Objects[1].(*widget.Label).Text != header {
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Label).SetText(chopAddr(Duels.Index[id].Duelist.Address))
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(Duels.Index[id].Duelist.getRankString())

					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[3].(*widget.Label).SetText(chopAddr(Duels.Index[id].Opponent.Address))
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[1].(*widget.Label).SetText(Duels.Index[id].Opponent.getRankString())

					o.(*fyne.Container).Objects[1].(*widget.Label).SetText(header)

					if Duels.Index[id].Items > 1 {
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 0)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 1)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[3].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 2)

						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0] = Duels.Index[id].Opponent.IconImage(0, 0)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[0] = Duels.Index[id].Opponent.IconImage(0, 1)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0] = Duels.Index[id].Opponent.IconImage(0, 2)
						o.Refresh()
						return
					}

					if Duels.Index[id].Items > 0 {
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 0)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 1)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[3].(*fyne.Container).Objects[0] = layout.NewSpacer()

						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0] = Duels.Index[id].Opponent.IconImage(0, 0)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[0] = Duels.Index[id].Opponent.IconImage(0, 1)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0] = layout.NewSpacer()
						o.Refresh()
						return
					}

					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 0)
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0] = layout.NewSpacer()
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[3].(*fyne.Container).Objects[0] = layout.NewSpacer()

					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0] = Duels.Index[id].Opponent.IconImage(0, 0)
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[0] = layout.NewSpacer()
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0] = layout.NewSpacer()
					o.Refresh()
				}
			}()
		})

	Ready.List.OnSelected = func(id widget.ListItemID) {
		if !loaded {
			return
		}
		if search.ready.searching {
			if search.ready.results == nil {
				return
			}
			selected_duel = search.ready.results[id]
		} else {
			if id+1 > len(Ready.All) {
				return
			}
			selected_duel = Ready.All[id]
		}

		Duels.RLock()
		defer Duels.RUnlock()
		if checkOwnerAndRefs() {
			var info dialog.Dialog
			ref_button := widget.NewButton("Ref Duel", func() {
				dialog.NewConfirm("Ref Duel", fmt.Sprintf("Would you like to Ref this Duel?\n\n%s", Duels.Index[selected_duel].dryRefDuel()), func(b bool) {
					if b {
						info.Hide()
						tx := Duels.Index[selected_duel].refDuel()
						go menu.ShowTxDialog("Ref Duel", "Duels", tx, 3*time.Second, d.Window)
						resetToTabs()
					}
				}, d.Window).Show()
			})
			ref_button.Importance = widget.HighImportance

			refund_button := widget.NewButton("Refund", func() {
				dialog.NewConfirm("Refund Duel", "Would you like to refund this Duel?", func(b bool) {
					if b {
						if n := strconv.FormatUint(selected_duel, 10); n != "" {
							info.Hide()
							tx := Refund(n)
							go menu.ShowTxDialog("Refund Duel", "Duels", tx, 3*time.Second, d.Window)
							resetToTabs()
						}
					}
				}, d.Window).Show()
			})
			refund_button.Importance = widget.HighImportance

			info = dialog.NewCustom("Owner Options", "Done", container.NewHBox(ref_button, refund_button), d.Window)
			info.SetOnClosed(func() {
				Ready.List.UnselectAll()
			})
			info.Show()

		} else if Duels.Index[selected_duel].checkDuelAddresses() {
			stamp := Duels.Index[selected_duel].Stamp + 172800
			if stamp <= time.Now().Unix() {
				dialog.NewConfirm("Cancel Duel", "Would you like to cancel this Duel?", func(b bool) {
					if b {
						if n := strconv.FormatUint(selected_join, 10); n != "" {
							tx := Refund(n)
							go menu.ShowTxDialog("Cancel Duel", "Duels", tx, 3*time.Second, d.Window)
							resetToTabs()
						}
					}
					Ready.List.UnselectAll()
					start_duel.Show()
				}, d.Window).Show()
			} else {
				avail := time.Unix(stamp, 0)
				info := dialog.NewInformation("Cancel Duel", fmt.Sprintf("This duel can be canceled in %s", formatDuration(time.Until(avail))), d.Window)
				info.SetOnClosed(func() {
					Ready.List.UnselectAll()
				})
				info.Show()

			}
		}
	}

	// list of current graves
	Graveyard.List = widget.NewList(
		func() int {
			return len(Graveyard.All)
		},
		func() fyne.CanvasObject {
			return container.NewBorder(
				dwidget.NewCenterLabel(""),
				dwidget.NewCenterLabel(""),
				nil,
				nil,
				container.NewCenter(container.NewHBox(iconSmall(nil, "", false), layout.NewSpacer())))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			go func() {
				var id uint64
				if search.graves.searching {
					if search.graves.results == nil {
						return
					}
					id = search.graves.results[i]
				} else {
					if i+1 > len(Graveyard.All) {
						return
					}
					id = Graveyard.All[i]
				}

				Graveyard.RLock()
				defer Graveyard.RUnlock()

				sc_label := fmt.Sprintf("SCID: %s", Graveyard.Index[id].Char)

				if Graveyard.Index[id].Char != "" {
					now := time.Now()
					avail := time.Unix(Graveyard.Index[id].Time, 0)

					var header string
					if now.Unix() >= avail.Unix() {
						header = fmt.Sprintf("Grave #%d   %s  -  Amount: (%s %s)   Available: (Yes)   Time in Grave: (%s)", id, menu.GetNFAName(Graveyard.Index[id].Char), rpc.FromAtomic(Graveyard.Index[id].findDiscount(), 5), Graveyard.Index[id].assetName(), formatDuration(time.Since(avail)))
					} else {
						left := time.Until(avail)
						header = fmt.Sprintf("Grave #%d   %s  -  Amount: (%s %s)   Available in: (%s)", id, menu.GetNFAName(Graveyard.Index[id].Char), rpc.FromAtomic(Graveyard.Index[id].findDiscount(), 5), Graveyard.Index[id].assetName(), formatDuration(left))
					}

					o.(*fyne.Container).Objects[1].(*widget.Label).SetText(header)
					o.(*fyne.Container).Objects[2].(*widget.Label).SetText(sc_label)

					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0] = Graveyard.Index[id].IconImage(0)
					o.Refresh()
				}
			}()
		})

	// Confirm a revive from the graveyard, if revive tx is confirmed will auto claim it
	accept_grave := widget.NewButton("Confirm", func() {
		Graveyard.RLock()
		defer Graveyard.RUnlock()
		scid := Graveyard.Index[selected_grave].Char
		tx := Graveyard.Index[selected_grave].Revive()
		go func() {
			go menu.ShowMessageDialog("Revive", fmt.Sprintf("TX: %s\n\nAuto claim tx will be sent once revive is confirmed", tx), 3*time.Second, d.Window)
			if rpc.ConfirmTx(tx, app_tag, 45) {
				if claim := rpc.ClaimNFA(scid); claim != "" {
					if rpc.ConfirmTx(claim, app_tag, 45) {
						d.Notification(app_tag, fmt.Sprintf("Claimed: %s", scid))
					}
				}
			}
		}()

		resetToTabs()
	})
	accept_grave.Importance = widget.HighImportance

	// Clicking on a grave will bring up revive confirmation
	Graveyard.List.OnSelected = func(id widget.ListItemID) {
		if search.graves.searching {
			if search.graves.results == nil {
				return
			}
			selected_grave = search.graves.results[id]
		} else {
			if id+1 > len(Graveyard.All) {
				return
			}
			selected_grave = Graveyard.All[id]
		}

		Graveyard.RLock()
		defer Graveyard.RUnlock()
		now := time.Now()
		avail := time.Unix(Graveyard.Index[selected_grave].Time, 0)

		if now.Unix() < avail.Unix() {
			info := dialog.NewInformation("Not Ready", fmt.Sprintf("This character can be revived from the graveyard in %s", formatDuration(time.Until(avail))), d.Window)
			info.SetOnClosed(func() {
				resetToTabs()
			})
			info.Show()

			return
		}

		start_duel.Hide()
		icon := Graveyard.Index[selected_grave].IconImage(1)
		revive_fee := fmt.Sprintf("%s %s", rpc.FromAtomic(Graveyard.Index[selected_grave].findDiscount(), 5), Graveyard.Index[selected_grave].assetName())

		graveyard_cont := container.NewBorder(
			container.NewVBox(
				container.NewCenter(container.NewVBox(
					dwidget.NewCanvasText(fmt.Sprintf("Revive for %s", revive_fee), 18, fyne.TextAlignCenter),
					canvas.NewLine(bundle.TextColor)))),
			container.NewVBox(
				dwidget.NewCenterLabel(fmt.Sprintf("Revive\n\n%s\n\nfrom grave yard for %s", Graveyard.Index[selected_grave].Char, revive_fee)),
				container.NewCenter(container.NewAdaptiveGrid(2, accept_grave, back_button))),
			nil,
			nil,
			container.NewCenter(icon))

		max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = graveyard_cont
		Graveyard.List.UnselectAll()
	}

	// List of final results
	Finals.List = widget.NewList(
		func() int {
			return len(Finals.All)
		},
		func() fyne.CanvasObject {
			return container.NewBorder(
				container.NewCenter(widget.NewRichText()),
				dwidget.NewCenterLabel(""),
				nil,
				nil,
				container.NewCenter( // 0
					container.NewHBox( // 0-0
						container.NewVBox( // 0-0-0
							container.NewHBox( // 0-0-0-0
								container.NewVBox( // 0-0-0-0-0
									dwidget.NewTrailingLabel("")), // 0-0-0-0-0-1
								container.NewStack(iconSmall(nil, "", false)), // 0-0-0-0-1
								container.NewStack(layout.NewSpacer()),        // 0-0-0-0-1
								container.NewStack(layout.NewSpacer())),       // 0-0-0-0-3
							dwidget.NewTrailingLabel(""),  // 0-0-0-1
							dwidget.NewTrailingLabel("")), // 0-0-0-2

						widget.NewSeparator(), // 0-0-1
						container.NewBorder(nil, dwidget.NewCenterLabel(""), nil, nil, container.NewCenter(container.NewStack(layout.NewSpacer()))), // 0-0-2
						widget.NewSeparator(), // 0-0-3

						container.NewVBox( // 0-0-4
							container.NewHBox( // 0-0-4-0
								container.NewStack(layout.NewSpacer()), // 0-0-4-0-0
								container.NewStack(layout.NewSpacer()), // 0-0-4-0-1
								container.NewStack(layout.NewSpacer()), // 0-0-4-0-2
								container.NewVBox( // 0-0-4-0-3
									widget.NewLabel(""))), // 0-0-4-0-3-0
							widget.NewLabel(""),    // 0-0-4-1
							widget.NewLabel(""))))) // 0-0-4-2
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			go func() {
				var id uint64
				if search.results.searching {
					if search.results.results == nil {
						return
					}
					id = search.results.results[i]
				} else {
					if i+1 > len(Finals.All) {
						return
					}
					id = Finals.All[i]
				}

				Duels.RLock()
				defer Duels.RUnlock()

				header := Duels.Index[id].resultsHeaderString()

				if Duels.Index[id].Complete && Duels.Index[id].Opponent.Char != "" && o.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*widget.RichText).String() != header {
					var arrow fyne.CanvasObject
					if Duels.Index[id].Odds < 475 {
						arrow = bundle.LeftArrow(fyne.NewSize(80, 80))
					} else {
						arrow = bundle.RightArrow(fyne.NewSize(80, 80))

					}
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0] = arrow
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[1].(*widget.Label).SetText(Duels.Index[id].endedIn())

					o.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*widget.RichText).ParseMarkdown(header)
					o.(*fyne.Container).Objects[2].(*widget.Label).SetText(fmt.Sprintf("Winner: %s %s", chopAddr(Duels.Index[id].Winner), Leaders.getRecordByAddress(Duels.Index[id].Winner)))

					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Label).SetText(Duels.Index[id].Duelist.findDuelResult())
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[3].(*fyne.Container).Objects[0].(*widget.Label).SetText(Duels.Index[id].Opponent.findDuelResult())

					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(Duels.Index[id].Duelist.getRankString())
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[1].(*widget.Label).SetText(Duels.Index[id].Opponent.getRankString())

					aEarn, bEarn := Duels.Index[id].findEarning()
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*widget.Label).SetText(fmt.Sprintf("Earnings: (%s %s)", rpc.FromAtomic(aEarn, 5), Duels.Index[id].assetName()))
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[2].(*widget.Label).SetText(fmt.Sprintf("Earnings: (%s %s)", rpc.FromAtomic(bEarn, 5), Duels.Index[id].assetName()))
					if Duels.Index[id].Items > 1 {
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 0)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 1)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[3].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 2)

						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0] = Duels.Index[id].Opponent.IconImage(0, 0)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[0] = Duels.Index[id].Opponent.IconImage(0, 1)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0] = Duels.Index[id].Opponent.IconImage(0, 2)
						o.Refresh()
						return
					}

					if Duels.Index[id].Items > 0 {
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 0)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 1)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[3].(*fyne.Container).Objects[0] = layout.NewSpacer()

						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0] = Duels.Index[id].Opponent.IconImage(0, 0)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[0] = Duels.Index[id].Opponent.IconImage(0, 1)
						o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0] = layout.NewSpacer()
						o.Refresh()
						return
					}

					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[0] = Duels.Index[id].Duelist.IconImage(0, 0)
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0] = layout.NewSpacer()
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[3].(*fyne.Container).Objects[0] = layout.NewSpacer()

					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0] = Duels.Index[id].Opponent.IconImage(0, 0)
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[0] = layout.NewSpacer()
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0] = layout.NewSpacer()
					o.Refresh()
				}
			}()
		})

	// Shows SCIDs of chars and items used
	Finals.List.OnSelected = func(id widget.ListItemID) {
		Duels.RLock()
		defer Duels.RUnlock()

		i := Finals.All[id]

		buildCont := func(s string) *fyne.Container {
			entry := widget.NewEntry()
			entry.SetText(s)

			copy := widget.NewButtonWithIcon("", dreams.FyneIcon("contentCopy"), func() { d.Window.Clipboard().SetContent(s) })
			copy.Importance = widget.LowImportance

			return container.NewBorder(nil, nil, nil, copy, entry)
		}

		var duelist_form, opponent_form []*widget.FormItem
		duelist_form = append(duelist_form, widget.NewFormItem("Duelist", layout.NewSpacer()))
		opponent_form = append(opponent_form, widget.NewFormItem("Opponent", layout.NewSpacer()))

		duelist_form = append(duelist_form, widget.NewFormItem("Character", buildCont(Duels.Index[i].Duelist.Char)))
		opponent_form = append(opponent_form, widget.NewFormItem("Character", buildCont(Duels.Index[i].Opponent.Char)))

		if Duels.Index[i].Items > 0 {
			duelist_form = append(duelist_form, widget.NewFormItem("Item 1", buildCont(Duels.Index[i].Duelist.Item1)))
			opponent_form = append(opponent_form, widget.NewFormItem("Item 1", buildCont(Duels.Index[i].Opponent.Item1)))
		}

		if Duels.Index[i].Items > 1 {
			duelist_form = append(duelist_form, widget.NewFormItem("Item 2", buildCont(Duels.Index[i].Duelist.Item2)))
			opponent_form = append(opponent_form, widget.NewFormItem("Item 2", buildCont(Duels.Index[i].Opponent.Item2)))
		}

		duelist_form = append(duelist_form, widget.NewFormItem("", dwidget.NewLine(100, 1, bundle.TextColor)))

		max := container.NewVBox(widget.NewForm(duelist_form...), widget.NewForm(opponent_form...))

		dia := dialog.NewCustom("SCIDs", "Done", max, d.Window)
		dia.Resize(fyne.NewSize(400, 0))
		dia.SetOnClosed(func() { Finals.List.UnselectAll() })
		dia.Show()
	}

	// Leader board list widget
	Leaders.list = widget.NewList(
		func() int {
			return len(Leaders.board)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(fmt.Sprintf("%s   %s", Leaders.board[i].address, Leaders.getRecordByIndex(i)))
			o.Refresh()
		})

	tabs = container.NewAppTabs(
		container.NewTabItemWithIcon("", ResourceDuelCirclePng, layout.NewSpacer()),
		container.NewTabItem("Join", container.NewBorder(nil, search.joins.searchDuels([]string{"Address", "Amount", "Currency", "Death Match"}, false, &Joins, d), nil, nil, Joins.List)),
		container.NewTabItem("Duels", container.NewBorder(nil, search.ready.searchDuels([]string{"Address", "Amount", "Currency", "Death Match"}, false, &Ready, d), nil, nil, Ready.List)),
		container.NewTabItem("Graves", container.NewBorder(nil, search.graves.searchGraves(d), nil, nil, Graveyard.List)),
		container.NewTabItem("Results", container.NewBorder(nil, search.results.searchDuels([]string{"Address", "Amount", "Currency", "Death Match", "My Duels", "Odds"}, true, &Finals, d), nil, nil, Finals.List)),
		container.NewTabItem("Leaders", Leaders.list))

	tabs.DisableIndex(0)
	tabs.SelectIndex(1)

	tabs.OnSelected = func(ti *container.TabItem) {
		switch ti.Text {
		case "Join":
			Joins.List.UnselectAll()
			selected_join = 0
		case "Duels":
			Ready.List.UnselectAll()
			selected_duel = 0
		case "Graves":
			Graveyard.List.UnselectAll()
			selected_grave = 0
		case "Results":
			Finals.List.UnselectAll()
		}
	}

	// Initialize dReams container stack
	D.LeftLabel = widget.NewLabel("")
	D.RightLabel = widget.NewLabel("")
	D.LeftLabel.SetText("Total Duels Held: ()      Ready Duels: ()")
	D.RightLabel.SetText("dReams Balance: " + rpc.DisplayBalance("dReams") + "      Dero Balance: " + rpc.DisplayBalance("Dero") + "      Height: " + rpc.Wallet.Display.Height)

	top_label := container.NewHBox(D.LeftLabel, layout.NewSpacer(), D.RightLabel)

	max = container.NewStack(container.NewHSplit(container.NewStack(sync_cont), container.NewStack(bundle.NewAlpha120(), tabs)))
	max.Objects[0].(*container.Split).SetOffset(0)

	// Start a duel form
	duel_curr := widget.NewSelect([]string{"DERO", "dReams"}, nil)
	duel_curr.PlaceHolder = "Currency:"

	duel_amt := dwidget.NewDeroEntry("", 0.1, 5)
	duel_amt.SetPlaceHolder("DERO:")

	// Death match options
	dm_label := dwidget.NewCenterLabel("")

	hc_label := dwidget.NewCenterLabel("If a higher ranked player joins your duel, you will get paid back odds on a loss")

	enable_dm := widget.NewRadioGroup([]string{"No", "Yes"}, nil)
	enable_dm.SetSelected("No")
	enable_dm.Horizontal = true
	enable_dm.Required = true
	enable_dm.OnChanged = func(s string) {
		if s == "Yes" {
			updateDMLabel()
			return
		}
		dm_label.SetText("")
	}

	updateDMLabel = func() {
		current_items := uint64(0)
		if Inventory.Item1.Select.Selected != "" {
			current_items++
		}
		if Inventory.Item2.Select.Selected != "" {
			current_items++
		}
		if enable_dm.Selected == "Yes" {
			dm_label.SetText(fmt.Sprintf("Death match enabled, any items used will be sent to the winner of this duel and loosing character will go to the graveyard\n\nCharacter revival from graveyard will initially be (%s %s)", rpc.FromAtomic((rpc.ToAtomic(duel_amt.Text, 5)*2)*(current_items+1), 5), duel_curr.Selected))
		}
	}

	enable_hc := widget.NewRadioGroup([]string{"No", "Yes"}, nil)
	enable_hc.SetSelected("No")
	enable_hc.Horizontal = true
	enable_hc.Required = true
	enable_hc.OnChanged = func(s string) {
		if s == "Yes" {
			hc_label.SetText("Hardcore mode enabled, winner of this duel will take the full pot regardless of rank")
			return
		}
		hc_label.SetText("If a higher ranked player joins your duel, you will get receive odds on a loss")
	}

	duel_curr.OnChanged = func(s string) {
		if enable_dm.Selected == "Yes" {
			updateDMLabel()
		}

		switch s {
		case "dReams":
			duel_amt.SetPlaceHolder("dReams:")
		default:
			duel_amt.SetPlaceHolder("DERO:")
		}
	}

	duel_amt.OnChanged = func(s string) {
		if enable_dm.Selected == "Yes" {
			updateDMLabel()
		}
	}

	opts := []string{"Leg", "Arm", "Chest", "Neck", "Head"}
	opt_select := widget.NewSelect(opts, nil)
	opt_select.PlaceHolder = "Aim for:"

	start_duel_char := widget.NewLabel("")
	start_duel_item1 := widget.NewLabel("")
	start_duel_item2 := widget.NewLabel("")

	start_duel_form := []*widget.FormItem{}
	start_duel_form = append(start_duel_form, widget.NewFormItem("Character", start_duel_char))
	start_duel_form = append(start_duel_form, widget.NewFormItem("Item 1", start_duel_item1))
	start_duel_form = append(start_duel_form, widget.NewFormItem("Item 2", start_duel_item2))
	start_duel_form = append(start_duel_form, widget.NewFormItem("", duel_curr))
	start_duel_form = append(start_duel_form, widget.NewFormItem("Amount", duel_amt))
	start_duel_form = append(start_duel_form, widget.NewFormItem("Death match", enable_dm))
	start_duel_form = append(start_duel_form, widget.NewFormItem("Hardcore", enable_hc))
	start_duel_form = append(start_duel_form, widget.NewFormItem("Aim for", opt_select))

	// Action confirmation button for joining and starting duels
	confirm_button := widget.NewButton("Confirm", func() {
		switch accepting_duel {
		case false:
			if Inventory.Item1.Select.SelectedIndex() >= 0 && Inventory.Item1.Select.Selected == Inventory.Item2.Select.Selected {
				info := dialog.NewInformation("Same Item", "You can't use the same item twice", d.Window)
				info.SetOnClosed(func() {
					Inventory.Item2.Select.FocusLost()
				})
				info.Show()
				Inventory.Item2.Select.FocusGained()
				return
			}

			if duel_curr.Selected == "" {
				info := dialog.NewInformation("Start Duel", "Choose a currency", d.Window)
				info.SetOnClosed(func() {
					duel_curr.FocusLost()
				})
				info.Show()
				duel_curr.FocusGained()
				return
			}

			f, err := strconv.ParseFloat(duel_amt.Text, 64)
			if err != nil || f == 0 {
				info := dialog.NewInformation("Start Duel", "Amount error", d.Window)
				info.SetOnClosed(func() {
					duel_amt.FocusLost()
				})
				info.Show()
				duel_amt.FocusGained()
				return
			}

			if opt_select.SelectedIndex() < 0 {
				info := dialog.NewInformation("Start Duel", "Choose aim option", d.Window)
				info.SetOnClosed(func() {
					opt_select.FocusLost()
				})
				info.Show()
				opt_select.FocusGained()
				return
			}

			start_duel.Hide()
			items := uint64(0)
			var item1_str, item2_str string
			char_str := asset_map[removeRank(Inventory.Character.Select.Selected)]
			if Inventory.Item1.Select.SelectedIndex() >= 0 {
				item1_str = asset_map[removeRank(Inventory.Item1.Select.Selected)]
				items++
			}
			if Inventory.Item2.Select.SelectedIndex() >= 0 {
				item2_str = asset_map[removeRank(Inventory.Item2.Select.Selected)]
				items++
			}

			rule := uint64(0)
			if enable_hc.Selected == "Yes" {
				rule = 1
			}

			dm := uint64(0)
			if enable_dm.Selected == "Yes" {
				dm = 1
			}

			aim := uint64(opt_select.SelectedIndex())

			if duel_curr.Selected != "DERO" && rpc.SCIDs[duel_curr.Selected] == "" {
				info := "Choose a currency"
				if duel_curr.Selected != "" {
					info = fmt.Sprintf("Error getting %s SCID", duel_curr.Selected)
				}
				dialog.NewInformation("Start Duel", info, d.Window).Show()
				resetToTabs()
				return
			}

			tx := StartDuel(rpc.ToAtomic(f, 5), items, rule, dm, aim, char_str, item1_str, item2_str, rpc.SCIDs[duel_curr.Selected])
			go menu.ShowTxDialog("Start Duel", "Duels", tx, 3*time.Second, d.Window)

			max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = tabs
			start_duel.Show()

		case true:
			if opt_select.SelectedIndex() < 0 {
				info := dialog.NewInformation("Accept Duel", "Choose aim option", d.Window)
				info.SetOnClosed(func() {
					opt_select.FocusLost()
				})
				info.Show()
				opt_select.FocusGained()
				return
			}

			start_duel.Hide()
			var item1_str, item2_str string
			items := uint64(0)
			char_str := asset_map[removeRank(Inventory.Character.Select.Selected)]
			if Inventory.Item1.Select.SelectedIndex() >= 0 {
				item1_str = asset_map[removeRank(Inventory.Item1.Select.Selected)]
				items++
			}
			if Inventory.Item2.Select.SelectedIndex() >= 0 {
				item2_str = asset_map[removeRank(Inventory.Item2.Select.Selected)]
				items++
			}

			tx := Duels.Index[selected_join].AcceptDuel(items, uint64(opt_select.SelectedIndex()), char_str, item1_str, item2_str)
			go menu.ShowTxDialog("Accept Duel", "Duels", tx, 3*time.Second, d.Window)

			accepting_duel = false
			max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = tabs
			start_duel.Show()
		}
		resetToTabs()
	})
	confirm_button.Importance = widget.HighImportance

	cancel_button := widget.NewButton("Cancel", func() {
		start_duel_char.SetText("")
		start_duel_item1.SetText("")
		start_duel_item2.SetText("")
		resetToTabs()
	})

	action_cont := container.NewCenter(container.NewAdaptiveGrid(2, confirm_button, cancel_button))

	start_label_spacer := canvas.NewRectangle(color.RGBA{0, 0, 0, 0})
	start_label_spacer.SetMinSize(fyne.NewSize(0, 120))

	start_duel.OnTapped = func() {
		if Inventory.Item1.Select.SelectedIndex() >= 0 && Inventory.Item1.Select.Selected == Inventory.Item2.Select.Selected {
			info := dialog.NewInformation("Same Item", "You can't use the same item twice", d.Window)
			info.SetOnClosed(func() {
				Inventory.Item2.Select.FocusLost()
			})
			info.Show()
			Inventory.Item2.Select.FocusGained()
			return
		}

		starting_duel = true
		start_duel_char.SetText(Inventory.Character.Select.Selected)

		start_duel_item1.SetText(Inventory.Item1.Select.Selected)
		if Inventory.Item1.Select.Selected == "" {
			start_duel_item1.SetText("None")
		}

		start_duel_item2.SetText(Inventory.Item2.Select.Selected)
		if Inventory.Item2.Select.Selected == "" {
			start_duel_item2.SetText("None")
		}

		max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = container.NewBorder(
			container.NewCenter(container.NewVBox(dwidget.NewCanvasText("Start Duel", 18, fyne.TextAlignCenter), canvas.NewLine(bundle.TextColor))),
			action_cont, nil,
			nil,
			container.NewBorder(layout.NewSpacer(), container.NewStack(start_label_spacer, container.NewVBox(dm_label, hc_label)), nil, nil, container.NewVBox(layout.NewSpacer(), container.NewCenter(widget.NewForm(start_duel_form...)), layout.NewSpacer())))
		start_duel.Hide()
		char_clear.Disable()
		item1_clear.Disable()
		item2_clear.Disable()

		Inventory.Character.Select.Disable()
		Inventory.Item1.Select.Disable()
		Inventory.Item2.Select.Disable()
	}

	// accept duel objects
	accept_form := []*widget.FormItem{}
	accept_form = append(accept_form, widget.NewFormItem("Aim for", opt_select))

	accept_duel.OnTapped = func() {
		accepting_duel = true
		max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = container.NewBorder(
			container.NewCenter(container.NewVBox(dwidget.NewCanvasText("Choose Your Aim", 18, fyne.TextAlignCenter), canvas.NewLine(bundle.TextColor))),
			action_cont,
			nil,
			nil,
			container.NewCenter(widget.NewForm(accept_form...)))
	}

	// Main duel process
	go func() {
		time.Sleep(3 * time.Second)
		var offset int
		var synced bool
		for {
			select {
			case <-d.Receive():
				if !rpc.Wallet.IsConnected() || !rpc.Daemon.IsConnected() {
					start_duel.Hide()
					total_rank_label.Text = ""
					character_cont.Objects[0].(*fyne.Container).Objects[0] = character
					item1_cont.Objects[1].(*fyne.Container).Objects[0] = item1
					item2_cont.Objects[1].(*fyne.Container).Objects[0] = item2

					max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = tabs

					sync_label.SetText("Connect to Daemon and Wallet to sync")
					max.Objects[0].(*container.Split).Leading.(*fyne.Container).Objects[0] = container.NewStack(sync_cont)

					Disconnected()
					Inventory.Character.ClearAll()
					Inventory.Item1.ClearAll()
					Inventory.Item2.ClearAll()
					synced = false
					loaded = false
					d.WorkDone()
					continue
				}

				if !synced && gnomes.Scan(d.IsConfiguring()) {
					sync_label.SetText("Creating duels index, this may take a few minutes to complete")
					logger.Println("[Duels] Syncing")
					gnomes.GetStorage("DUELBUCKET", "DUELS", &Duels)
					synced = true
				} else {
					sync_label.SetText("Waiting for Gnomon to sync")
				}

				if synced {
					if !loaded {
						if r, ok := rpc.GetStringKey(DUELSCID, "rds", rpc.Daemon.Rpc).(float64); ok {
							sync_prog.Max = r + 1
						} else {
							sync_prog.Max = 50
						}
					}

					if GetJoins() {
						Joins.List.Refresh()
					}

					if !loaded {
						sync_prog.SetValue(0)
						sync_prog.Max = float64(len(Duels.Index))
						sync_label.SetText("Matching opponents, this may take a few minutes to complete")
					}

					if GetAllDuels() {
						Ready.List.Refresh()
					}

					if !loaded {
						sync_prog.SetValue(0)
						sync_prog.Max = float64(len(Duels.Index)) + 1
						sync_label.SetText("Getting results, this may take a few minutes to complete")
					}

					if GetFinals() {
						Finals.List.Refresh()
					}

					if !loaded {
						sync_prog.SetValue(0)
						sync_label.SetText("Getting graves, this may take a few minutes to complete")
						if gnomon.IsReady() {
							info := gnomon.GetAllSCIDVariableDetails(DUELSCID)
							sync_prog.Max = float64(len(info)) + 1
						} else {
							sync_prog.Max = float64(400)
						}

					}

					GetGraveyard()
					if offset%10 == 0 {
						if !gnomon.IsClosing() {
							gnomes.StoreBolt("DUELBUCKET", "DUELS", &Duels)
						}
					}

					if !loaded {
						sync_label.SetText("")
						sync_prog.Max = 1
						sync_prog.SetValue(0)
						sync_prog.Refresh()
						loaded = true
						max.Objects[0].(*container.Split).Leading.(*fyne.Container).Objects[0] = container.NewStack(equip_cont)
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
	}()

	return container.NewBorder(dwidget.LabelColor(top_label), nil, nil, nil, max)
}

// Update ui progress bar when syncing
func updateSyncProgress(bar *widget.ProgressBar) {
	if bar != nil && bar.Max > 1 {
		if bar.Value < bar.Max {
			bar.SetValue(bar.Value + 1)
		}
	}
}

// Search bar widget for duels
func (s *searches) searchDuels(opts []string, complete bool, l *dwidget.Lists, d *dreams.AppObject) (max *fyne.Container) {
	amt_entry := dwidget.NewDeroEntry("", 0.1, 5)
	amt_entry.SetPlaceHolder("Amount:")

	yn_select := widget.NewSelect([]string{"Yes", "No"}, nil)

	curr_select := widget.NewSelect([]string{"DERO", "dReams"}, nil)
	curr_select.PlaceHolder = "Select Currency"

	search_entry := widget.NewEntry()
	search_entry.SetPlaceHolder("Search:")
	search_select := widget.NewSelect(opts, func(s string) {
		switch s {
		case "Address":
			max.Objects[0] = search_entry
			search_entry.SetPlaceHolder(s + ":")
		case "Amount":
			amt_entry.AllowFloat = true
			max.Objects[0] = amt_entry
		case "Currency":
			max.Objects[0] = curr_select
		case "Death Match":
			max.Objects[0] = yn_select
		case "My Duels":
			max.Objects[0] = search_entry
			search_entry.SetPlaceHolder(s + ":")
			search_entry.SetText(rpc.Wallet.Address)
		case "Odds":
			amt_entry.AllowFloat = false
			max.Objects[0] = amt_entry
		case "SCID":
			max.Objects[0] = search_entry
			search_entry.SetPlaceHolder(s + ":")
		}
	})

	clear_button := widget.NewButtonWithIcon("", dreams.FyneIcon("searchReplace"), func() {
		search_entry.SetText("")
		s.results = nil
		s.searching = false
		search_select.Enable()
		l.List.Length = func() int {
			return len(l.All)
		}
	})
	clear_button.Importance = widget.LowImportance

	search_button := widget.NewButton("Search", func() {
		s.results = nil
		switch search_select.Selected {
		case "Address":
			if len(search_entry.Text) != 66 {
				dialog.NewInformation("Search", "Not a valid address", d.Window).Show()
				return
			}
			for u, r := range Duels.Index {
				if r.Duelist.Address == search_entry.Text || r.Opponent.Address == search_entry.Text {
					if r.Complete == complete {
						s.results = append(s.results, u)
					}
				}
			}
		case "Amount":
			search_amt, err := strconv.ParseFloat(amt_entry.Text, 64)
			if err != nil {
				dialog.NewInformation("Search", "Error parsing amount value", d.Window).Show()
				return
			}
			for u, r := range Duels.Index {
				if r.Amt == rpc.ToAtomic(search_amt, 5) && r.Complete == complete {
					s.results = append(s.results, u)
				}
			}
		case "Currency":
			for u, r := range Duels.Index {
				if r.assetName() == curr_select.Selected && r.Complete == complete {
					s.results = append(s.results, u)
				}
			}
		case "Death Match":
			for u, r := range Duels.Index {
				if r.DM == yn_select.Selected && r.Complete == complete {
					s.results = append(s.results, u)
				}
			}
		case "My Duels":
			for u, r := range Duels.Index {
				if r.Duelist.Address == rpc.Wallet.Address || r.Opponent.Address == rpc.Wallet.Address {
					s.results = append(s.results, u)
				}
			}
		case "Odds":
			search_odds, err := strconv.ParseUint(amt_entry.Text, 10, 64)
			if err != nil {
				dialog.NewInformation("Search", "Error parsing odds value", d.Window).Show()
				return
			}
			for u, r := range Duels.Index {
				if r.Odds == search_odds {
					s.results = append(s.results, u)
				}
			}
		default:
			info := dialog.NewInformation("Search", "Not a valid search query", d.Window)
			info.SetOnClosed(func() {
				search_select.FocusLost()
			})
			info.Show()
			search_select.FocusGained()
			return
		}

		if s.results == nil {
			dialog.NewInformation("Search", "No results found", d.Window).Show()
			s.searching = false
			search_select.Enable()
			return
		}

		sort.Slice(s.results, func(i, j int) bool { return s.results[i] < s.results[j] })

		s.searching = true
		search_select.Disable()

		l.List.Length = func() int {
			return len(s.results)
		}
		l.List.Refresh()
	})

	max = container.NewBorder(
		nil,
		nil,
		container.NewBorder(nil, nil, clear_button, nil, search_select),
		search_button,
		search_entry)

	return
}

// Search bar widget for graveyard
func (s *searches) searchGraves(d *dreams.AppObject) (max *fyne.Container) {
	amt_entry := dwidget.NewDeroEntry("", 0.1, 5)
	amt_entry.SetPlaceHolder("Amount:")
	amt_entry.AllowFloat = true

	time_select := widget.NewSelect([]string{"Available", "Coming soon"}, nil)

	curr_select := widget.NewSelect([]string{"DERO", "dReams"}, nil)
	curr_select.PlaceHolder = "Select Currency"

	search_entry := widget.NewEntry()
	search_entry.SetPlaceHolder("Search:")
	search_select := widget.NewSelect([]string{"Amount", "Availability", "Currency", "SCID"}, func(s string) {
		switch s {
		case "Amount":
			max.Objects[0] = amt_entry
		case "Availability":
			max.Objects[0] = time_select
		case "Currency":
			max.Objects[0] = curr_select
		case "SCID":
			max.Objects[0] = search_entry
			search_entry.SetPlaceHolder(s + ":")
		}
	})

	search_finals_clear := widget.NewButtonWithIcon("", dreams.FyneIcon("searchReplace"), func() {
		s.results = nil
		s.searching = false
		search_select.Enable()
		Graveyard.List.Length = func() int {
			return len(Graveyard.All)
		}
	})
	search_finals_clear.Importance = widget.LowImportance

	search_finals_button := widget.NewButton("Search", func() {
		s.results = nil
		switch search_select.Selected {
		case "Amount":
			search_amt, err := strconv.ParseFloat(amt_entry.Text, 64)
			if err != nil {
				dialog.NewInformation("Search", "Error parsing amount value", d.Window).Show()
				return
			}
			for u, r := range Graveyard.Index {
				if r.Amt == rpc.ToAtomic(search_amt, 5) {
					s.results = append(s.results, u)
				}
			}
		case "Availability":
			for u, r := range Graveyard.Index {
				switch time_select.Selected {
				case "Available":
					if int64(r.Time) < time.Now().Unix() {
						s.results = append(s.results, u)
					}
				case "Coming soon":
					if int64(r.Time) > time.Now().Unix() {
						s.results = append(s.results, u)
					}
				}
			}
		case "Currency":
			for u, r := range Graveyard.Index {
				if r.assetName() == curr_select.Selected {
					s.results = append(s.results, u)
				}
			}

		case "SCID":
			if len(search_entry.Text) != 64 {
				dialog.NewInformation("Search", "Not a valid SCID", d.Window).Show()
				return
			}
			for u, r := range Graveyard.Index {
				if r.Char == search_entry.Text {
					s.results = append(s.results, u)
				}
			}
		default:
			info := dialog.NewInformation("Search", "Not a valid search query", d.Window)
			info.SetOnClosed(func() {
				search_select.FocusLost()
			})
			info.Show()
			search_select.FocusGained()
			return
		}

		if s.results == nil {
			dialog.NewInformation("Search", "No results found", d.Window).Show()
			return
		}

		sort.Slice(s.results, func(i, j int) bool { return s.results[i] < s.results[j] })

		s.searching = true
		search_select.Disable()

		Graveyard.List.Length = func() int {
			return len(s.results)
		}
		Graveyard.List.Refresh()
	})

	return container.NewBorder(
		nil,
		nil,
		container.NewBorder(nil, nil, search_finals_clear, nil, search_select),
		search_finals_button,
		search_entry)
}
