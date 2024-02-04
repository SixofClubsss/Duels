package duel

import (
	"fmt"
	"image/color"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/blang/semver/v4"
	dreams "github.com/dReam-dApps/dReams"
	"github.com/dReam-dApps/dReams/bundle"
	"github.com/dReam-dApps/dReams/dwidget"
	"github.com/dReam-dApps/dReams/gnomes"
	"github.com/dReam-dApps/dReams/menu"
	"github.com/dReam-dApps/dReams/rpc"
	"github.com/sirupsen/logrus"
)

const app_tag = "Duels"

var version = semver.MustParse("0.1.1-dev.1")
var gnomon = gnomes.NewGnomes()

// Check duel package version
func Version() semver.Version {
	return version
}

// Start Asset Duels as a stand alone app to be locally ran or imported and ran
func StartApp() {
	n := runtime.NumCPU()
	runtime.GOMAXPROCS(n)
	gnomes.InitLogrusLog(logrus.InfoLevel)
	config := menu.ReadDreamsConfig(app_tag)

	a := app.NewWithID(fmt.Sprintf("%s Client", app_tag))
	a.Settings().SetTheme(bundle.DeroTheme(config.Skin))
	w := a.NewWindow(app_tag)
	w.SetIcon(ResourceDuelIconPng)
	w.Resize(fyne.NewSize(1400, 800))
	w.CenterOnScreen()
	w.SetMaster()
	done := make(chan struct{})

	menu.Theme.Img = *canvas.NewImageFromResource(menu.DefaultThemeResource())
	d := dreams.AppObject{
		App:        a,
		Window:     w,
		Background: container.NewStack(&menu.Theme.Img),
	}

	closeFunc := func() {
		menu.SetClose(true)
		save := dreams.SaveData{
			Skin:   config.Skin,
			DBtype: gnomon.DBStorageType(),
			Theme:  menu.Theme.Name,
		}

		if rpc.Daemon.Rpc == "" {
			save.Daemon = config.Daemon
		} else {
			save.Daemon = []string{rpc.Daemon.Rpc}
		}

		menu.WriteDreamsConfig(save)
		gnomon.Stop(app_tag)
		d.StopProcess()
		w.Close()
	}

	w.SetCloseIntercept(closeFunc)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println()
		closeFunc()
	}()

	gnomon.SetDBStorageType("boltdb")
	gnomon.SetFastsync(true, true, 10000)
	d.SetChannels(1)

	// Create dwidget rpc connect box
	connect_box := dwidget.NewHorizontalEntries(app_tag, 1)
	connect_box.Button.OnTapped = func() {
		// Get Dero wallet address
		rpc.GetAddress(app_tag)

		// Ping daemon
		rpc.Ping()

		// Start Gnomon with search filters when connected to daemon
		if rpc.Daemon.IsConnected() && !gnomon.IsInitialized() && !gnomon.IsStarting() {
			go gnomes.StartGnomon(app_tag, gnomon.DBStorageType(), []string{rpc.GetSCCode(DUELSCID), gnomes.NFA_SEARCH_FILTER}, 0, 0, nil)
		}
	}

	// Main routine
	go func() {
		synced := false
		time.Sleep(3 * time.Second)
		ticker := time.NewTicker(3 * time.Second)

		for {
			select {
			case <-ticker.C: // do on interval
				rpc.Ping()
				rpc.EchoWallet(app_tag)
				go rpc.GetDreamsBalances(rpc.SCIDs)
				rpc.GetWalletHeight(app_tag)
				gnomes.EndPoint()

				if rpc.Wallet.IsConnected() && gnomon.IsReady() {
					connect_box.RefreshBalance()
					menu.DisableIndexControls(false)
					if !synced {
						checkNFAs(app_tag, true, false, nil)
						synced = true
					}
				} else {
					menu.Assets.Asset = []menu.Asset{}
					menu.Assets.List.Refresh()
					menu.Assets.Claim.Hide()
					menu.DisableIndexControls(true)
					synced = false
				}

				if gnomon.IsRunning() {
					gnomon.IndexContains()
					menu.Info.RefreshIndexed()
				}

				if gnomon.HasIndex(100) {
					gnomon.Synced(true)
					gnomon.Checked(true)
				} else {
					gnomon.Synced(false)
					gnomon.Checked(false)
					synced = false
				}

				d.SignalChannel()

			case <-d.Closing(): // exit loop
				logger.Printf("[%s] Closing...\n", app_tag)
				ticker.Stop()
				d.CloseAllDapps()
				time.Sleep(time.Second)
				done <- struct{}{}
				return

			}
		}
	}()

	// Gnomon shutdown on daemon disconnect
	connect_box.Disconnect.OnChanged = func(b bool) {
		if !b {
			gnomon.Stop(app_tag)
		}
	}

	// Set any saved daemon configs
	connect_box.AddDaemonOptions(config.Daemon)

	// Adding dReams indicator panel for wallet, daemon and Gnomon
	connect_box.Container.Objects[0].(*fyne.Container).Add(menu.StartIndicators())

	tabs := container.NewAppTabs(
		container.NewTabItem("Duels", LayoutAllItems(menu.Assets.SCIDs, &d)),
		container.NewTabItem("Assets", menu.PlaceAssets(app_tag, profile(&d), nil, bundle.ResourceMarketIconPng, &d)),
		container.NewTabItem("Market", menu.PlaceMarket(&d)),
		container.NewTabItem("Log", rpc.SessionLog(app_tag, version)))

	tabs.SetTabLocation(container.TabLocationBottom)

	go func() {
		time.Sleep(450 * time.Millisecond)
		w.SetContent(container.NewStack(d.Background, tabs, container.NewVBox(layout.NewSpacer(), connect_box.Container)))
	}()
	w.ShowAndRun()
	<-done
	logger.Printf("[%s] Closed\n", app_tag)
}

// Checks for valid duel NFAs
//   - all true clears and syncs menu/duel lists, false does only duel lists
func checkNFAs(tag string, all, progress bool, scids map[string]string) {
	if gnomon.IsReady() {
		if scids == nil {
			scids = gnomon.GetAllOwnersAndSCIDs()
		}

		Inventory.ClearAll()
		if all {
			menu.Assets.Asset = []menu.Asset{}
		}

		logger.Printf("[%s] Checking NFA Assets\n", tag)
		if progress {
			sync_prog.Max = float64(len(scids))
			sync_prog.SetValue(0)
		}

		for sc := range scids {
			if !rpc.Wallet.IsConnected() || !gnomon.IsRunning() {
				break
			}
			if progress {
				updateSyncProgress(sync_prog)
			}
			checkNFAOwner(sc, all)
		}

		Inventory.SortAll()
		if all {
			menu.Assets.SortList()
			menu.Assets.List.Refresh()
		}
	}
}

// Checks for valid NFA owner and adds items to inventory, used only in stand alone version
func checkNFAOwner(scid string, all bool) {
	if gnomon.IsRunning() {
		if header, _ := gnomon.GetSCIDValuesByKey(scid, "nameHdr"); header != nil {
			owner, _ := gnomon.GetSCIDValuesByKey(scid, "owner")
			file, _ := gnomon.GetSCIDValuesByKey(scid, "fileURL")
			collection, _ := gnomon.GetSCIDValuesByKey(scid, "collection")
			icon, _ := gnomon.GetSCIDValuesByKey(scid, "iconURLHdr")
			if owner != nil && file != nil && collection != nil && icon != nil {
				if owner[0] == rpc.Wallet.Address && menu.ValidNFA(file[0]) {
					var add menu.Asset
					add.Name = header[0]
					add.Collection = collection[0]
					add.SCID = scid
					add.Type = menu.AssetType(collection[0], "typeHdr")

					if isValidCharacter(collection[0]) || isValidItem(collection[0]) {
						if all {
							menu.Assets.Add(add, icon[0])
						}
						AddItemsToInventory(scid, header[0], owner[0], collection[0])
					}
				}
			}
		}
	}
}

// User profile layout with dreams.AssetSelects
func profile(d *dreams.AppObject) fyne.CanvasObject {
	line := canvas.NewLine(bundle.TextColor)
	form := []*widget.FormItem{}
	form = append(form, widget.NewFormItem("Name", menu.NameEntry()))
	form = append(form, widget.NewFormItem("", layout.NewSpacer()))
	form = append(form, widget.NewFormItem("", container.NewVBox(line)))
	form = append(form, widget.NewFormItem("Theme", menu.ThemeSelect(d)))
	form = append(form, widget.NewFormItem("", layout.NewSpacer()))
	form = append(form, widget.NewFormItem("", container.NewVBox(line)))

	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(450, 0))

	return container.NewCenter(container.NewBorder(spacer, nil, nil, nil, widget.NewForm(form...)))
}
