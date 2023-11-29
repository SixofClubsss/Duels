package duel

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"github.com/blang/semver/v4"
	dreams "github.com/dReam-dApps/dReams"
	"github.com/dReam-dApps/dReams/bundle"
	"github.com/dReam-dApps/dReams/dwidget"
	"github.com/dReam-dApps/dReams/menu"
	"github.com/dReam-dApps/dReams/rpc"
	"github.com/sirupsen/logrus"
)

const app_tag = "Duels"

var version = semver.MustParse("0.1.0-dev")

// Check duel package version
func Version() semver.Version {
	return version
}

// Start Asset Duels as a stand alone app to be locally ran or imported and ran
func StartApp() {
	n := runtime.NumCPU()
	runtime.GOMAXPROCS(n)
	menu.InitLogrusLog(logrus.InfoLevel)
	config := menu.ReadDreamsConfig(app_tag)

	a := app.NewWithID(fmt.Sprintf("%s Client", app_tag))
	a.Settings().SetTheme(bundle.DeroTheme(config.Skin))
	w := a.NewWindow(app_tag)
	w.SetIcon(resourceDuelIconPng)
	w.Resize(fyne.NewSize(1400, 800))
	w.SetMaster()
	done := make(chan struct{})

	dreams.Theme.Img = *canvas.NewImageFromResource(nil)
	d := dreams.AppObject{
		App:        a,
		Window:     w,
		Background: container.NewStack(&dreams.Theme.Img),
	}

	closeFunc := func() {
		menu.CloseAppSignal(true)
		save := dreams.SaveData{
			Skin:   config.Skin,
			DBtype: menu.Gnomes.DBType,
		}

		if rpc.Daemon.Rpc == "" {
			save.Daemon = config.Daemon
		} else {
			save.Daemon = []string{rpc.Daemon.Rpc}
		}

		menu.WriteDreamsConfig(save)
		menu.Gnomes.Stop(app_tag)
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

	menu.Gnomes.DBType = "boltdb"
	menu.Gnomes.Fast = true
	rpc.InitBalances()
	d.SetChannels(1)

	menu.Assets.Asset_map = make(map[string]string)

	// Create dwidget rpc connect box
	connect_box := dwidget.NewHorizontalEntries(app_tag, 1)
	connect_box.Button.OnTapped = func() {
		// Get Dero wallet address
		rpc.GetAddress(app_tag)

		// Ping daemon
		rpc.Ping()

		// Start Gnomon with search filters when connected to daemon
		if rpc.Daemon.IsConnected() && !menu.Gnomes.IsInitialized() && !menu.Gnomes.Start {
			go menu.StartGnomon(app_tag, menu.Gnomes.DBType, []string{rpc.GetSCCode(DUELSCID), menu.NFA_SEARCH_FILTER}, 0, 0, nil)
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
				menu.GnomonEndPoint()

				if rpc.Wallet.IsConnected() && menu.Gnomes.IsReady() {
					connect_box.RefreshBalance()
					menu.DisableIndexControls(false)
					if !synced {
						checkNFAs(app_tag, synced, nil)
						synced = true
					}
				} else {
					menu.Assets.Assets = []string{}
					menu.Assets.Asset_list.Refresh()
					menu.Control.Claim_button.Hide()
					menu.DisableIndexControls(true)
					synced = false
				}

				if menu.Gnomes.IsRunning() {
					menu.Gnomes.IndexContains()
					menu.Assets.Gnomes_index.Text = (" Indexed SCIDs: " + strconv.Itoa(int(menu.Gnomes.SCIDS)))
					menu.Assets.Gnomes_index.Refresh()
				} else {
					menu.Assets.Gnomes_index.Text = (" Indexed SCIDs: 0")
					menu.Assets.Gnomes_index.Refresh()
				}

				if menu.Gnomes.HasIndex(100) {
					menu.Gnomes.Synced(true)
					menu.Gnomes.Checked(true)
				} else {
					menu.Gnomes.Synced(false)
					menu.Gnomes.Checked(false)
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
			menu.Gnomes.Stop(app_tag)
		}
	}

	// Set any saved daemon configs
	connect_box.AddDaemonOptions(config.Daemon)

	// Adding dReams indicator panel for wallet, daemon and Gnomon
	connect_box.Container.Objects[0].(*fyne.Container).Add(menu.StartIndicators())

	tabs := container.NewAppTabs(
		container.NewTabItem("Duels", LayoutAllItems(menu.Assets.Asset_map, &d)),
		container.NewTabItem("Assets", menu.PlaceAssets(app_tag, nil, bundle.ResourceMarketIconPng, w)),
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

// Checks for valid duel NFAs, used only in stand alone version
func checkNFAs(tag string, gc bool, scids map[string]string) {
	if menu.Gnomes.IsReady() && !gc {
		if scids == nil {
			scids = menu.Gnomes.GetAllOwnersAndSCIDs()
		}

		menu.Assets.Assets = []string{}
		logger.Printf("[%s] Checking NFA Assets\n", tag)

		for sc := range scids {
			if !rpc.Wallet.IsConnected() || !menu.Gnomes.IsRunning() {
				break
			}

			checkNFAOwner(sc)
		}

		sort.Strings(menu.Assets.Assets)
		menu.Assets.Asset_list.Refresh()
		Inventory.SortAll()
	}
}

// Checks for valid NFA owner and adds items to inventory, used only in stand alone version
func checkNFAOwner(scid string) {
	if menu.Gnomes.IsRunning() {
		if header, _ := menu.Gnomes.GetSCIDValuesByKey(scid, "nameHdr"); header != nil {
			owner, _ := menu.Gnomes.GetSCIDValuesByKey(scid, "owner")
			file, _ := menu.Gnomes.GetSCIDValuesByKey(scid, "fileURL")
			collection, _ := menu.Gnomes.GetSCIDValuesByKey(scid, "collection")
			if owner != nil && file != nil && collection != nil {
				if owner[0] == rpc.Wallet.Address && menu.ValidNFA(file[0]) {
					if collection[0] == "Dero Desperados" {
						menu.Assets.Add(header[0], scid)
						AddItemsToInventory(scid, header[0], owner[0], collection[0])
					} else if collection[0] == "Desperado Guns" {
						menu.Assets.Add(header[0], scid)
						AddItemsToInventory(scid, header[0], owner[0], collection[0])
					} else if collection[0] == "High Strangeness" {
						menu.Assets.Add(header[0], scid)
						AddItemsToInventory(scid, header[0], owner[0], collection[0])
					} else if collection[0] == "Death By Cupcake" {
						menu.Assets.Add(header[0], scid)
						AddItemsToInventory(scid, header[0], owner[0], collection[0])
					}
				}
			}
		}
	}
}
