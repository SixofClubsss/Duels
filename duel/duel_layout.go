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

// Layout all duel items
func LayoutAllItems(asset_map map[string]string, d *dreams.AppObject) fyne.CanvasObject {
	selected_join := uint64(0)
	selected_duel := uint64(0)
	selected_grave := uint64(0)
	var resetToTabs, updateDMLabel func()
	var starting_duel, accepting_duel bool
	var max *fyne.Container
	var tabs *container.AppTabs

	// Character and item selection objects
	sil := canvas.NewImageFromResource(resourceCowboySilPng)
	sil.SetMinSize(fyne.NewSize(150, 320))

	total_rank_label := dwidget.NewCanvasText("", 15, fyne.TextAlignCenter)
	start_duel := widget.NewButton("Start a Duel", nil)
	start_duel.Hide()

	character := canvas.NewImageFromResource(bundle.ResourceAvatarFramePng)
	character.SetMinSize(fyne.NewSize(100, 100))
	char_opts := []string{}
	Inventory.Character.Select = widget.NewSelect(char_opts, nil)
	Inventory.Character.Select.PlaceHolder = "Character:"
	char_clear := widget.NewButtonWithIcon("", fyne.Theme.Icon(fyne.CurrentApp().Settings().Theme(), "viewRefresh"), nil)
	character_cont := container.NewBorder(
		container.NewBorder(nil, nil, char_clear, nil, Inventory.Character.Select),
		nil,
		nil,
		nil,
		container.NewCenter(character))

	char_clear.OnTapped = func() {
		Inventory.Character.Select.ClearSelected()
		Inventory.Character.Select.Selected = ""
	}

	Inventory.Character.Select.OnChanged = func(s string) {
		go func() {
			total_rank_label.Text = fmt.Sprintf("You're Rank: (R%d)", Inventory.findRank())
			total_rank_label.Refresh()
			if s == "" {
				Inventory.Item1.Select.Disable()
				Inventory.Item2.Select.Disable()
				Inventory.Item1.Select.Selected = ""
				Inventory.Item2.Select.Selected = ""
				start_duel.Hide()
				character_cont.Objects[0] = container.NewCenter(character)
				if starting_duel {
					resetToTabs()
					info := dialog.NewInformation("Start Duel", "Choose a character to duel with", d.Window)
					info.SetOnClosed(func() {
						Inventory.Character.Select.FocusLost()
					})
					info.Show()
					Inventory.Character.Select.FocusGained()

				}
				return
			}

			if selected_join == 0 {
				start_duel.Show()
			}
			Inventory.Item1.Select.Enable()
			Inventory.RLock()
			character_cont.Objects[0] = container.NewCenter(iconLarge(Inventory.characters[removeRank(s)].img, s))
			Inventory.RUnlock()
		}()
	}

	select_spacer := canvas.NewRectangle(color.RGBA{0, 0, 0, 0})
	select_spacer.SetMinSize(fyne.NewSize(120, 0))

	item1 := canvas.NewImageFromResource(bundle.ResourceAvatarFramePng)
	item1.SetMinSize(fyne.NewSize(100, 100))
	item1_opts := []string{}
	Inventory.Item1.Select = widget.NewSelect(item1_opts, nil)
	Inventory.Item1.Select.PlaceHolder = "Item 1:"
	item1_clear := widget.NewButtonWithIcon("", fyne.Theme.Icon(fyne.CurrentApp().Settings().Theme(), "viewRefresh"), nil)

	item1_cont := container.NewVBox(container.NewBorder(nil, nil, item1_clear, nil, container.NewMax(select_spacer, Inventory.Item1.Select)), container.NewCenter(item1))
	item1_clear.OnTapped = func() {
		Inventory.Item1.Select.ClearSelected()
		Inventory.Item1.Select.Selected = ""
	}

	Inventory.Item1.Select.OnChanged = func(s string) {
		updateDMLabel()
		total_rank_label.Text = fmt.Sprintf("You're Rank: (R%d)", Inventory.findRank())
		total_rank_label.Refresh()
		if s == "" {
			Inventory.Item2.Select.Disable()
			Inventory.Item2.Select.Selected = ""
			Inventory.Item2.Select.Options = Inventory.Item1.Select.Options
			Inventory.Item2.Select.Refresh()
			item1_cont.Objects[1] = container.NewCenter(item1)
			if Inventory.Character.Select.Selected == "" {
				Inventory.Item1.Select.Disable()
			} else {
				Inventory.Item1.Select.Enable()
			}
			return
		}

		Inventory.RLock()
		item1_cont.Objects[1] = container.NewCenter(iconLarge(Inventory.items[removeRank(s)].img, s))
		Inventory.RUnlock()
		Inventory.Item2.Select.Enable()
	}
	Inventory.Item1.Select.Disable()

	item2 := canvas.NewImageFromResource(bundle.ResourceAvatarFramePng)
	item2.SetMinSize(fyne.NewSize(100, 100))
	item2_opts := []string{}
	Inventory.Item2.Select = widget.NewSelect(item2_opts, nil)
	Inventory.Item2.Select.PlaceHolder = "Item 2:"
	item2_clear := widget.NewButtonWithIcon("", fyne.Theme.Icon(fyne.CurrentApp().Settings().Theme(), "viewRefresh"), nil)
	item2_cont := container.NewVBox(container.NewBorder(nil, nil, item2_clear, nil, container.NewMax(select_spacer, Inventory.Item2.Select)), container.NewCenter(item2))
	item2_clear.OnTapped = func() {
		Inventory.Item2.Select.ClearSelected()
		Inventory.Item2.Select.Selected = ""
	}

	Inventory.Item2.Select.OnChanged = func(s string) {
		updateDMLabel()
		total_rank_label.Text = fmt.Sprintf("You're Rank: (R%d)", Inventory.findRank())
		total_rank_label.Refresh()
		if s == "" {
			if Inventory.Character.Select.Selected == "" && rpc.IsReady() {
				Inventory.Item1.Select.Enable()
			}
			item2_cont.Objects[1] = container.NewCenter(item2)
			return
		}

		Inventory.RLock()
		item2_cont.Objects[1] = container.NewCenter(iconLarge(Inventory.items[removeRank(s)].img, s))
		Inventory.RUnlock()

		Inventory.Item1.Select.Disable()
	}
	Inventory.Item2.Select.Disable()

	equip_alpha := canvas.NewRectangle(color.RGBA{0, 0, 0, 0})
	equip_alpha.SetMinSize(fyne.NewSize(450, 650))

	equip_box := container.NewCenter(equip_alpha,
		container.NewMax(container.NewBorder(container.NewMax(character_cont), container.NewBorder(total_rank_label, nil, nil, nil, container.NewMax()), container.NewMax(item1_cont), container.NewMax(item2_cont), sil)))

	options_select := widget.NewSelect([]string{"Recheck Assets", "Claim All", "Clear Cache"}, nil)
	options_select.PlaceHolder = "Options:"
	options_select.OnChanged = func(s string) {
		switch s {
		case "Recheck Assets":
			if rpc.IsReady() {
				dialog.NewConfirm("Recheck Assets", "Would you like to recheck wallet for Duel assets?", func(b bool) {
					if b {
						Inventory.ClearAll()
						character_cont.Objects[0] = container.NewCenter(character)
						item1_cont.Objects[1] = container.NewCenter(item1)
						item2_cont.Objects[1] = container.NewCenter(item2)
						checkNFAs("Duels", false, nil)
					}
				}, d.Window).Show()
			} else {
				dialog.NewInformation("Recheck Assets", "You are not connected to daemon or wallet", d.Window).Show()
			}
		case "Claim All":
			if rpc.IsReady() {
				claimable := checkClaimable()
				l := len(claimable)
				if l > 0 {
					dialog.NewConfirm("Claim All", fmt.Sprintf("Claim your %d available assets?", l), func(b bool) {
						if b {
							go claimClaimable(claimable, d)
						}
					}, d.Window).Show()
				} else {
					dialog.NewInformation("Claim All", "You have no claimable assets", d.Window).Show()
				}
			} else {
				dialog.NewInformation("Claim All", "You are not connected to daemon or wallet", d.Window).Show()
			}
		case "Clear Cache":
			if menu.Gnomes.DBType == "boltdb" {
				dialog.NewConfirm("Clear Image Cache", "Would you like to clear your stored image cache?", func(b bool) {
					if b {
						deleteIndex()
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

	equip_cont := container.NewBorder(
		dwidget.NewCanvasText("Your Inventory", 18, fyne.TextAlignCenter),
		container.NewAdaptiveGrid(2, container.NewMax(start_duel), options_select),
		nil,
		nil,
		equip_box)

	// Opponent box
	opponent_sil := canvas.NewImageFromResource(resourceCowboySilPng)
	opponent_sil.SetMinSize(fyne.NewSize(150, 320))

	opponent_character := canvas.NewImageFromResource(bundle.ResourceAvatarFramePng)
	opponent_character.SetMinSize(fyne.NewSize(100, 100))
	opponent_character_cont := container.NewCenter(opponent_character)

	opponent_item1 := canvas.NewImageFromResource(bundle.ResourceAvatarFramePng)
	opponent_item1.SetMinSize(fyne.NewSize(100, 100))

	opponent_item1_cont := container.NewVBox(container.NewCenter(opponent_item1))

	opponent_item2 := canvas.NewImageFromResource(bundle.ResourceAvatarFramePng)
	opponent_item2.SetMinSize(fyne.NewSize(100, 100))
	opponent_item2_cont := container.NewVBox(container.NewCenter(opponent_item2))

	opponent_alpha := canvas.NewRectangle(color.RGBA{0, 0, 0, 0})
	opponent_alpha.SetMinSize(fyne.NewSize(450, 500))

	opponent_label := widget.NewLabel("")
	opponent_label.Alignment = fyne.TextAlignCenter

	opponent_equip_box := container.NewCenter(opponent_alpha,
		container.NewBorder(opponent_character_cont, layout.NewSpacer(), opponent_item1_cont, opponent_item2_cont, opponent_sil))

	accept_duel := widget.NewButton("Accept Duel", nil)

	resetToTabs = func() {
		selected_join = 0
		selected_duel = 0
		selected_grave = 0
		starting_duel = false
		accepting_duel = false
		max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = tabs
		max.Objects[0].(*container.Split).Trailing.Refresh()
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

	opponent_equip_cont := container.NewBorder(nil, container.NewAdaptiveGrid(2, accept_duel, back_button), nil, nil, opponent_equip_box)

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
				o.(*fyne.Container).Objects[2].(*fyne.Container).Objects[1].(*widget.Label).SetText(fmt.Sprintf("Duelist: %s %s", Duels.Index[id].Duelist.Address, Leaders.getRecordByAddress(Duels.Index[id].Duelist.Address)))

				if Duels.Index[id].Num != "" && !Duels.Index[id].Complete {
					o.(*fyne.Container).Objects[2].(*fyne.Container).Objects[0].(*widget.Label).SetText(Duels.Index[id].Duelist.getRankString())
					o.(*fyne.Container).Objects[1].(*widget.Label).SetText(fmt.Sprintf("Duel #%s   Rank: (R%d)   Amount: (%s %s)   Items: (%d)   Death Match: (%s)   Hardcore: (%s)", Duels.Index[id].Num, Duels.Index[id].getDuelistRank(), rpc.FromAtomic(Duels.Index[id].Amt, 5), Duels.Index[id].assetName(), Duels.Index[id].Items, Duels.Index[id].DM, Duels.Index[id].Rule))

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
			if rpc.Wallet.Address == Duels.Index[selected_join].Duelist.Address {
				dialog.NewConfirm("Cancel Duel", "Would you like to cancel this Duel?", func(b bool) {
					if b {
						if n := strconv.FormatUint(selected_join, 10); n != "" {
							Refund(n)
							resetToTabs()
						}
					}
					Joins.List.UnselectAll()
					start_duel.Show()

				}, d.Window).Show()

				return
			}

			if !Duels.Index[selected_join].validateCollection() {
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

				if Inventory.Item1.Select.Selected == Inventory.Item2.Select.Selected {
					info := dialog.NewInformation("Same Item", "You can't use the same item twice", d.Window)
					info.SetOnClosed(func() {
						item2_clear.FocusLost()
					})
					info.Show()
					item2_clear.FocusGained()
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

			character_rank_label := dwidget.NewCenterLabel(fmt.Sprintf("{R%d}", Duels.Index[selected_join].Duelist.getCharacterRank()))
			char_cont := container.NewCenter(
				container.NewBorder(
					nil,
					character_rank_label,
					nil,
					nil,
					Duels.Index[selected_join].Duelist.IconImage(1, 0)))

			opponent_item1_cont = container.NewCenter()
			opponent_item2_cont = container.NewCenter()

			if items > 0 {
				icon := Duels.Index[selected_join].Duelist.IconImage(1, 1)
				item_rank_label := dwidget.NewCenterLabel(fmt.Sprintf("{R%d}", Duels.Index[selected_join].Duelist.getItemRank(0)))
				opponent_item1_cont = container.NewCenter(
					container.NewBorder(
						item_rank_label,
						nil,
						nil,
						nil,
						icon))
			}

			if items > 1 {
				icon := Duels.Index[selected_join].Duelist.IconImage(1, 2)
				item_rank_label := dwidget.NewCenterLabel(fmt.Sprintf("{R%d}", Duels.Index[selected_join].Duelist.getItemRank(1)))
				opponent_item2_cont = container.NewCenter(
					container.NewBorder(
						item_rank_label,
						nil,
						nil,
						nil,
						icon))
			}

			r2 := Inventory.findRank()
			perc, r1, diff := Duels.Index[selected_join].diffOdds(r2)

			opponent_equip_box = container.NewCenter(opponent_alpha,
				container.NewBorder(
					char_cont,
					widget.NewLabel(fmt.Sprintf("Opponent Rank: (R%d)   Death match: (%s)   Hardcore: (%s)", r1, Duels.Index[selected_join].DM, Duels.Index[selected_join].Rule)),
					opponent_item1_cont,
					opponent_item2_cont,
					container.NewCenter(opponent_sil)))

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

			opponent_equip_cont = container.NewBorder(dwidget.NewCanvasText(fmt.Sprintf("Accept %s for %s %s", header, rpc.FromAtomic(amt, 5), asset_name), 18, fyne.TextAlignCenter), container.NewVBox(opponent_label, container.NewAdaptiveGrid(2, accept_duel, back_button)), nil, nil, opponent_equip_box)

			max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = container.NewMax(bundle.NewAlpha180(), opponent_equip_cont)
			max.Objects[0].(*container.Split).Trailing.Refresh()
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
								dwidget.NewCenterLabel(""),                  // 0-0-0-0-0
								container.NewMax(iconSmall(nil, "", false)), // 0-0-0-0-1
								container.NewMax(layout.NewSpacer()),        // 0-0-0-0-2
								container.NewMax(layout.NewSpacer())),       // 0-0-0-0-3
							dwidget.NewTrailingLabel("")), // 0-0-0-1

						widget.NewSeparator(),                        // 0-0-1
						canvas.NewText("   VS   ", bundle.TextColor), // 0-0-2
						widget.NewSeparator(),                        // 0-0-3

						container.NewVBox( // 0-0-4
							container.NewHBox( // 0-0-4-0
								container.NewMax(layout.NewSpacer()), // 0-0-4-0-0
								container.NewMax(layout.NewSpacer()), // 0-0-4-0-1
								container.NewMax(layout.NewSpacer()), // 0-0-4-0-2
								dwidget.NewCenterLabel("")),          // 0-0-4-0-3
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
				if Duels.Index[id].Opponent.Char != "" {
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Label).SetText(chopAddr(Duels.Index[id].Duelist.Address))
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(Duels.Index[id].Duelist.getRankString())

					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[0].(*fyne.Container).Objects[3].(*widget.Label).SetText(chopAddr(Duels.Index[id].Opponent.Address))
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[4].(*fyne.Container).Objects[1].(*widget.Label).SetText(Duels.Index[id].Opponent.getRankString())

					o.(*fyne.Container).Objects[1].(*widget.Label).SetText(fmt.Sprintf("Duel #%s   Pot: (%s %s)   Items: (%d)   Death Match: (%s)", Duels.Index[id].Num, rpc.FromAtomic(Duels.Index[id].Amt*2, 5), Duels.Index[id].assetName(), Duels.Index[id].Items, Duels.Index[id].DM))

					o.(*fyne.Container).Objects[2].(*widget.Label).SetText(fmt.Sprintf("Ready for: %v", Duels.Index[id].readySince()))

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
			info = dialog.NewCustom("Owner Options", "Done", container.NewHBox(widget.NewButton("Ref Duel", func() {
				dialog.NewConfirm("Ref Duel", fmt.Sprintf("Would you like to Ref this Duel?\n\n%s", Duels.Index[selected_duel].dryRefDuel()), func(b bool) {
					if b {
						info.Hide()
						Duels.Index[selected_duel].refDuel()

						resetToTabs()
					}
				}, d.Window).Show()
			}),
				widget.NewButton("Refund", func() {
					dialog.NewConfirm("Refund Duel", "Would you like to refund this Duel?", func(b bool) {
						if b {
							if n := strconv.FormatUint(selected_duel, 10); n != "" {
								info.Hide()
								Refund(n)
								resetToTabs()
							}
						}
					}, d.Window).Show()

				})), d.Window)
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
							Refund(n)
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
				if Graveyard.Index[id].Char != "" {
					now := time.Now()
					avail := time.Unix(Graveyard.Index[id].Time, 0)

					var text string
					if now.Unix() >= avail.Unix() {
						text = fmt.Sprintf("Grave #%d   %s  -  Amount: (%s %s)   Available: (Yes)   Time in Grave: (%s)", id, menu.GetNFAName(Graveyard.Index[id].Char), rpc.FromAtomic(Graveyard.Index[id].findDiscount(), 5), Graveyard.Index[id].assetName(), formatDuration(time.Since(avail)))
					} else {
						left := time.Until(avail)
						text = fmt.Sprintf("Grave #%d   %s  -  Amount: (%s %s)   Available in: (%s)", id, menu.GetNFAName(Graveyard.Index[id].Char), rpc.FromAtomic(Graveyard.Index[id].findDiscount(), 5), Graveyard.Index[id].assetName(), formatDuration(left))
					}

					o.(*fyne.Container).Objects[1].(*widget.Label).SetText(text)
					o.(*fyne.Container).Objects[2].(*widget.Label).SetText(fmt.Sprintf("SCID: %s", Graveyard.Index[id].Char))

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
		if tx := Graveyard.Index[selected_grave].Revive(); tx != "" {
			go func() {
				menu.ShowTxDialog("Revive", fmt.Sprintf("TX: %s\n\nAuto claim tx will be sent once revive is confirmed", tx), tx, 5*time.Second, d.Window)
				if rpc.ConfirmTx(tx, app_tag, 60) {
					if claim := rpc.ClaimNFA(scid); claim != "" {
						if rpc.ConfirmTx(claim, app_tag, 60) {
							d.Notification(app_tag, fmt.Sprintf("Claimed: %s", scid))
						}
					}
				}
			}()
		}

		resetToTabs()
	})

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
				dwidget.NewCanvasText("Revive for", 18, fyne.TextAlignCenter),
				dwidget.NewCanvasText(revive_fee, 18, fyne.TextAlignCenter)),
			container.NewVBox(
				dwidget.NewCenterLabel(fmt.Sprintf("Revive\n\n%s\n\nfrom grave yard for %s", Graveyard.Index[selected_grave].Char, revive_fee)),
				container.NewAdaptiveGrid(2, accept_grave, back_button)),
			nil,
			nil,
			container.NewCenter(icon))

		max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = graveyard_cont
		max.Objects[0].(*container.Split).Trailing.Refresh()
		Graveyard.List.UnselectAll()
	}

	// List of final results
	Finals.List = widget.NewList(
		func() int {
			return len(Finals.All)
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
								container.NewVBox( // 0-0-0-0-0
									dwidget.NewTrailingLabel("")), // 0-0-0-0-0-1
								container.NewMax(iconSmall(nil, "", false)), // 0-0-0-0-1
								container.NewMax(layout.NewSpacer()),        // 0-0-0-0-1
								container.NewMax(layout.NewSpacer())),       // 0-0-0-0-3
							dwidget.NewTrailingLabel(""),  // 0-0-0-1
							dwidget.NewTrailingLabel("")), // 0-0-0-2

						widget.NewSeparator(), // 0-0-1
						container.NewBorder(nil, dwidget.NewCenterLabel(""), nil, nil, container.NewCenter(container.NewMax(layout.NewSpacer()))), // 0-0-2
						widget.NewSeparator(), // 0-0-3

						container.NewVBox( // 0-0-4
							container.NewHBox( // 0-0-4-0
								container.NewMax(layout.NewSpacer()), // 0-0-4-0-0
								container.NewMax(layout.NewSpacer()), // 0-0-4-0-1
								container.NewMax(layout.NewSpacer()), // 0-0-4-0-2
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
				if Duels.Index[id].Complete && Duels.Index[id].Opponent.Char != "" {
					var arrow fyne.CanvasObject
					if Duels.Index[id].Odds < 475 {
						arrow = bundle.LeftArrow(fyne.NewSize(80, 80))
					} else {
						arrow = bundle.RightArrow(fyne.NewSize(80, 80))

					}
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0] = arrow
					o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container).Objects[1].(*widget.Label).SetText(Duels.Index[id].endedIn())

					o.(*fyne.Container).Objects[1].(*widget.Label).SetText(Duels.Index[id].resultsHeaderString())
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

	Finals.List.OnSelected = func(id widget.ListItemID) {
		// Doing nothing here for now

		// playAnimation(Duels.Index[Finals.All[id]].Winner, max, tabs)
		// resetToTabs()
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
		container.NewTabItem("Join", container.NewBorder(nil, search.joins.searchDuels([]string{"Address", "Amount", "Currency", "Death Match", "My Duels"}, false, &Joins, d), nil, nil, Joins.List)),
		container.NewTabItem("Duels", container.NewBorder(nil, search.ready.searchDuels([]string{"Address", "Amount", "Currency", "Death Match", "My Duels"}, false, &Ready, d), nil, nil, Ready.List)),
		container.NewTabItem("Graves", container.NewBorder(nil, search.graves.searchGraves(d), nil, nil, Graveyard.List)),
		container.NewTabItem("Results", container.NewBorder(nil, search.results.searchDuels([]string{"Address", "Amount", "Currency", "Death Match", "My Duels", "Odds"}, true, &Finals, d), nil, nil, Finals.List)),
		container.NewTabItem("Leaders", Leaders.list))

	tabs.OnSelected = func(ti *container.TabItem) {
		switch ti.Text {
		case "Join":
			selected_duel = 0
		case "Duels":
			Ready.List.UnselectAll()
			selected_duel = 0
		}
	}

	// Initialize dReams container stack
	D.LeftLabel = widget.NewLabel("")
	D.RightLabel = widget.NewLabel("")
	D.LeftLabel.SetText("Total Duels Held: ()      Ready Duels: ()")
	D.RightLabel.SetText("dReams Balance: " + rpc.DisplayBalance("dReams") + "      Dero Balance: " + rpc.DisplayBalance("Dero") + "      Height: " + rpc.Wallet.Display.Height)

	top_label := container.NewHBox(D.LeftLabel, layout.NewSpacer(), D.RightLabel)

	max = container.NewMax(container.NewHSplit(container.NewCenter(equip_cont), container.NewMax(bundle.NewAlpha120(), tabs)))
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
				dialog.NewInformation("Same Item", "You can't use the same item twice", d.Window).Show()
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

			if opt_select.SelectedIndex() < 0 {
				info := dialog.NewInformation("Start Duel", "Choose aim option", d.Window)
				info.SetOnClosed(func() {
					opt_select.FocusLost()
				})
				info.Show()
				opt_select.FocusGained()
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

			StartDuel(rpc.ToAtomic(f, 5), items, rule, dm, aim, char_str, item1_str, item2_str, rpc.SCIDs[duel_curr.Selected])

			max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = tabs
			max.Objects[0].(*container.Split).Trailing.Refresh()
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

			Duels.Index[selected_join].AcceptDuel(items, uint64(opt_select.SelectedIndex()), char_str, item1_str, item2_str)

			accepting_duel = false
			max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = tabs
			max.Objects[0].(*container.Split).Trailing.Refresh()
			start_duel.Show()
		}
		resetToTabs()
	})

	cancel_button := widget.NewButton("Cancel", func() {
		start_duel_char.SetText("")
		start_duel_item1.SetText("")
		start_duel_item2.SetText("")
		resetToTabs()
	})

	action_cont := container.NewAdaptiveGrid(2, confirm_button, cancel_button)

	start_label_spacer := canvas.NewRectangle(color.RGBA{0, 0, 0, 0})
	start_label_spacer.SetMinSize(fyne.NewSize(0, 120))

	start_duel.OnTapped = func() {
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

		max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = container.NewBorder(dwidget.NewCanvasText("Start Duel", 18, fyne.TextAlignCenter), action_cont, nil, nil, container.NewBorder(layout.NewSpacer(), container.NewMax(start_label_spacer, container.NewVBox(dm_label, hc_label)), nil, nil, container.NewVBox(layout.NewSpacer(), container.NewCenter(widget.NewForm(start_duel_form...)), layout.NewSpacer())))
		max.Objects[0].(*container.Split).Trailing.Refresh()
		start_duel.Hide()
	}

	// accept duel objects
	accept_form := []*widget.FormItem{}
	accept_form = append(accept_form, widget.NewFormItem("Aim for", opt_select))

	accept_duel.OnTapped = func() {
		accepting_duel = true
		max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = container.NewBorder(dwidget.NewCanvasText("Choose Your Aim", 18, fyne.TextAlignCenter), action_cont, nil, nil, container.NewCenter(widget.NewForm(accept_form...)))
		max.Objects[0].(*container.Split).Trailing.Refresh()
	}

	disconnectFunc := func() {
		start_duel.Hide()
		character_cont.Objects[0] = container.NewCenter(character)
		item1_cont.Objects[1] = container.NewCenter(item1)
		item2_cont.Objects[1] = container.NewCenter(item2)
	}

	go fetch(d, disconnectFunc)
	go func() {
		for !menu.ClosingApps() {
			if !rpc.IsReady() {
				max.Objects[0].(*container.Split).Trailing.(*fyne.Container).Objects[1] = tabs
				max.Objects[0].(*container.Split).Trailing.Refresh()
			}
			time.Sleep(time.Second)
		}
	}()

	return container.NewBorder(dwidget.LabelColor(top_label), nil, nil, nil, max)
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

	clear_button := widget.NewButtonWithIcon("", fyne.Theme.Icon(fyne.CurrentApp().Settings().Theme(), "searchReplace"), func() {
		search_entry.SetText("")
		s.results = nil
		s.searching = false
		search_select.Enable()
		l.List.Length = func() int {
			return len(l.All)
		}
	})

	search_button := widget.NewButton("Search", func() {
		s.results = nil
		switch search_select.Selected {
		case "Address":
			if len(search_entry.Text) != 64 {
				dialog.NewInformation("Search", "Not a valid address", d.Window).Show()
				return
			}
			for u, r := range Duels.Index {
				if r.Duelist.Address == "s" || r.Opponent.Address == "" {
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

	search_finals_clear := widget.NewButtonWithIcon("", fyne.Theme.Icon(fyne.CurrentApp().Settings().Theme(), "searchReplace"), func() {
		s.results = nil
		s.searching = false
		search_select.Enable()
		Graveyard.List.Length = func() int {
			return len(Graveyard.All)
		}
	})

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
