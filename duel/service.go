package duel

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/dReam-dApps/dReams/gnomes"
	"github.com/dReam-dApps/dReams/rpc"
	"github.com/docopt/docopt-go"
	"github.com/sirupsen/logrus"
)

var command_line string = `RefService
App to run RefService as a single process, powered by Gnomon and dReams.

Usage:
  RefService [options]
  RefService -h | --help

Options:
  -h --help                      Show this screen.
  --daemon=<127.0.0.1:10102>     Set daemon rpc address to connect.
  --wallet=<127.0.0.1:10103>     Set wallet rpc address to connect.
  --login=<user:pass>     	 Wallet rpc user:pass for auth.
  --fastsync=<true>	         Gnomon option,  true/false value to define loading at chain height on start up.
  --num-parallel-blocks=<5>      Gnomon option,  defines the number of parallel blocks to index.`

type service struct {
	Init       bool
	Processing bool
	sync.RWMutex
}

var Service service

// Start RefService
func (s *service) Start() {
	s.Lock()
	s.Init = true
	s.Unlock()
}

// Stop RefService
func (s *service) Stop() {
	s.Lock()
	s.Init = false
	s.Unlock()
}

// Check if RefService is running
func (s *service) IsRunning() bool {
	s.RLock()
	defer s.RUnlock()

	return s.Init
}

// Set RefService processing value
func (s *service) SetProcessing(b bool) {
	s.Lock()
	s.Processing = false
	s.Unlock()
}

// Check if RefService is currently processing
func (s *service) IsProcessing() bool {
	s.RLock()
	defer s.RUnlock()

	return s.Processing
}

// Ensure RefService is shutdown on app close
func (s *service) IsStopped() {
	s.Lock()
	defer s.Unlock()

	s.Init = false
	for s.Processing {
		logger.Println("[RefService] Waiting for service to close")
		time.Sleep(3 * time.Second)
	}
}

// Start RefService process with flags
func RunRefService() {
	n := runtime.NumCPU()
	runtime.GOMAXPROCS(n)

	gnomes.InitLogrusLog(logrus.InfoLevel)

	arguments, err := docopt.ParseArgs(command_line, nil, Version().String())
	if err != nil {
		logger.Fatalf("Error while parsing arguments: %s\n", err)
	}

	fastsync := true
	if arguments["--fastsync"] != nil {
		if arguments["--fastsync"].(string) == "false" {
			fastsync = false
		}
	}

	parallel := 5
	if arguments["--num-parallel-blocks"] != nil {
		s := arguments["--num-parallel-blocks"].(string)
		switch s {
		case "1":
			parallel = 1
		case "2":
			parallel = 2
		case "3":
			parallel = 3
		case "4":
			parallel = 4
		case "5":
			parallel = 5
		default:
			parallel = 5
		}
	}

	// Set default rpc params
	rpc.Daemon.Rpc = "127.0.0.1:10102"
	rpc.Wallet.Rpc = "127.0.0.1:10103"

	if arguments["--daemon"] != nil {
		if arguments["--daemon"].(string) != "" {
			rpc.Daemon.Rpc = arguments["--daemon"].(string)
		}
	}

	if arguments["--wallet"] != nil {
		if arguments["--wallet"].(string) != "" {
			rpc.Wallet.Rpc = arguments["--wallet"].(string)
		}
	}

	if arguments["--login"] != nil {
		if arguments["--login"].(string) != "" {
			rpc.Wallet.UserPass = arguments["--login"].(string)
		}
	}

	gnomon.SetFastsync(fastsync, true, 10000)
	gnomon.SetParallel(parallel)
	gnomon.SetDBStorageType("boltdb")

	logger.Println("[RefService]", version, runtime.GOOS, runtime.GOARCH)

	// Check for daemon connection
	rpc.Ping()
	if !rpc.Daemon.Connect {
		logger.Fatalf("[RefService] Daemon %s not connected\n", rpc.Daemon.Rpc)
	}

	// Check for wallet connection
	rpc.GetAddress("RefService")
	if !rpc.Wallet.Connect {
		logger.Fatalf("[RefService] Wallet %s not connected\n", rpc.Wallet.Rpc)
	}

	// Handle ctrl-c close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println()
		gnomon.Stop("RefService")
		rpc.Wallet.Connected(false)
		Service.Stop()
		for Service.IsProcessing() {
			logger.Println("[RefService] Waiting for service to close")
			time.Sleep(3 * time.Second)
		}
		logger.Println("[RefService] Closing")
		os.Exit(0)
	}()

	// Set up Gnomon search filters for Duel
	filter := []string{rpc.GetSCCode(DUELSCID), gnomes.NFA_SEARCH_FILTER}

	// Start Gnomon with search filters
	go gnomes.StartGnomon("RefService", gnomon.DBStorageType(), filter, 0, 0, nil)

	// Routine for checking daemon, wallet connection and Gnomon sync
	go func() {
		for !gnomon.IsInitialized() {
			time.Sleep(time.Second)
		}

		logger.Println("[RefService] Starting when Gnomon is synced")
		for gnomon.IsRunning() && rpc.IsReady() {
			rpc.Ping()
			rpc.EchoWallet("RefService")
			gnomon.IndexContains()
			if gnomon.GetLastHeight() >= gnomon.GetChainHeight()-3 && gnomon.HasIndex(1) {
				gnomon.Synced(true)
			} else {
				gnomon.Synced(false)
				if !gnomon.IsStarting() && gnomon.IsInitialized() {
					diff := gnomon.GetChainHeight() - gnomon.GetLastHeight()
					if diff > 3 {
						logger.Printf("[RefService] Gnomon has %d blocks to go\n", diff)
					}
				}
			}
			time.Sleep(3 * time.Second)
		}
	}()

	// Wait for Gnomon to sync
	for !gnomon.IsSynced() {
		time.Sleep(time.Second)
	}

	time.Sleep(time.Second)

	// Start RefService
	gnomes.GetStorage("DUELBUCKET", "DUELS", &Duels)
	Service.Start()
	refService()
}

// Main RefService process
func refService() {
	if rpc.IsReady() {
		logger.Println("[refService] Initializing")
		for i := 5; i > 0; i-- {
			if !Service.IsRunning() {
				break
			}
			logger.Println("[refService] Starting in", i)
			time.Sleep(time.Second)
		}

		if Service.IsRunning() {
			logger.Println("[refService] Starting")
		}

		for Service.IsRunning() && rpc.IsReady() {
			Service.SetProcessing(true)
			refGetJoins()
			refGetAllDuels()
			GetFinals()
			processReady()
			logger.Debugln("[refService] Joins:", len(Joins.All), Joins.All, "Ready:", len(Ready.All), Ready.All, "Finals:", len(Finals.All), Finals.All)

			if !gnomon.IsClosing() {
				gnomes.StoreBolt("DUELBUCKET", "DUELS", &Duels)
			}

			for i := 0; i < 10; i++ {
				time.Sleep(time.Second)
				if !Service.IsRunning() || !rpc.IsReady() {
					break
				}
			}
		}
		Service.SetProcessing(false)
		logger.Println("[refService] Shutting down")

		logger.Println("[refService] Done")
	}
	Service.Stop()
}

// Gets joinable duels for RefService
func refGetJoins() {
	if gnomon.IsReady() {
		_, initValue := gnomon.GetSCIDValuesByKey(DUELSCID, "init")
		if initValue != nil {
			if _, rounds := gnomon.GetSCIDValuesByKey(DUELSCID, "rds"); rounds != nil {
				Duels.Total = int(rounds[0])
			}

			u := uint64(0)
			for {
				u++
				if u > initValue[0] {
					break
				}

				if !rpc.Wallet.IsConnected() || !gnomon.IsReady() {
					break
				}

				e := Duels.SingleEntry(u)

				n := strconv.Itoa(int(u))
				_, init := gnomon.GetSCIDValuesByKey(DUELSCID, "init_"+n)
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
					logger.Debugln("[refGetJoins] Making")

					_, buffer := gnomon.GetSCIDValuesByKey(DUELSCID, "stamp_"+n)
					if buffer == nil {
						logger.Debugf("[refGetJoins] %s no start stamp\n", n)
						buffer = append(buffer, 0)
					}

					address, _ := gnomon.GetSCIDValuesByKey(DUELSCID, "own_a_"+n)
					if address == nil {
						logger.Debugf("[refGetJoins] %s no address\n", n)
						continue
					}

					if address[0] != rpc.Wallet.Address && time.Now().Unix() <= int64(buffer[0]) {
						logger.Debugf("[refGetJoins] %s in buffer\n", n)
						continue
					}

					_, items := gnomon.GetSCIDValuesByKey(DUELSCID, "items_"+n)
					if items == nil {
						logger.Debugf("[refGetJoins] %s no items\n", n)
						continue
					}

					deathmatch := "No"
					_, dm := gnomon.GetSCIDValuesByKey(DUELSCID, "dm_"+n)
					if dm == nil {
						logger.Debugf("[refGetJoins] %s no dm\n", n)
						continue
					}

					if dm[0] == 1 {
						deathmatch = "Yes"
					}

					hardcore := "No"
					_, rule := gnomon.GetSCIDValuesByKey(DUELSCID, "rule_"+n)
					if rule == nil {
						logger.Debugf("[refGetJoins] %s no rule\n", n)
						continue
					}

					if rule[0] == 1 {
						hardcore = "Yes"
					}

					_, amt := gnomon.GetSCIDValuesByKey(DUELSCID, "amt_"+n)
					if amt == nil {
						logger.Debugf("[refGetJoins] %s no amt\n", n)
						continue
					}

					_, option := gnomon.GetSCIDValuesByKey(DUELSCID, "op_a_"+n)
					if option == nil {
						logger.Debugf("[refGetJoins] %s no optA\n", n)
						continue
					}

					charA, _ := gnomon.GetSCIDValuesByKey(DUELSCID, "ch_a_"+n)
					if charA == nil {
						logger.Debugf("[refGetJoins] %s no charA\n", n)
						continue
					}

					token, _ := gnomon.GetSCIDValuesByKey(DUELSCID, "tkn_"+n)
					if token == nil {
						logger.Debugf("[refGetJoins] %s no token\n", n)
						token = append(token, "")
					}

					var item1Str, item2Str string
					if items[0] >= 1 {
						item1, _ := gnomon.GetSCIDValuesByKey(DUELSCID, "i1_a_"+n)
						if item1 == nil {
							logger.Debugf("[refGetJoins] %s should be a item1\n", n)
							continue
						}

						item1Str = item1[0]
					}

					if items[0] == 2 {
						item2, _ := gnomon.GetSCIDValuesByKey(DUELSCID, "i2_a_"+n)
						if item2 == nil {
							logger.Debugf("[refGetJoins] %s should be a item2\n", n)
							continue
						}

						item2Str = item2[0]
					}

					logger.Debugln("[refGetJoins] Storing A", n)
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
						},
					}

					if !e.validateCollection(false) {
						logger.Warnln("[refGetJoins] Not a valid duelist, refunding")
						retry := 0
						tx := Refund(n)
						time.Sleep(time.Second)
						retry += rpc.ConfirmTxRetry(tx, "refService", 60)
						continue
					}
					Duels.WriteEntry(u, e)
					Joins.All = append(Joins.All, u)
				} else if e.Opponent.Icon.Char == nil && !Joins.ExistsIndex(u) {
					Joins.All = append(Joins.All, u)
				}
			}
		}
	}

	Joins.SortIndex(false)
	Ready.SortIndex(false)
}

// Gets all duels for RefService
func refGetAllDuels() {
	if gnomon.IsReady() {
		for u, v := range Duels.Index {
			if !rpc.Wallet.IsConnected() || !gnomon.IsReady() {
				break
			}

			if v.Opponent.Char != "" {
				if Ready.ExistsIndex(u) {
					logger.Debugf("[refGetAllDuels] %d b Char already here\n", u)
				} else if !v.Complete {
					Ready.All = append(Ready.All, u)
				}

				Joins.RemoveIndex(u)

				continue
			}

			n := strconv.Itoa(int(u))
			if _, init := gnomon.GetSCIDValuesByKey(DUELSCID, "init_"+n); init != nil {
				address, _ := gnomon.GetSCIDValuesByKey(DUELSCID, "own_b_"+n)
				if address == nil {
					logger.Debugf("[refGetAllDuels] %s no address B\n", n)
					continue
				}

				_, ready_stamp := gnomon.GetSCIDValuesByKey(DUELSCID, "ready_"+n)
				if ready_stamp == nil {
					logger.Debugf("[refGetAllDuels] %s no ready stamp\n", n)
					ready_stamp = append(ready_stamp, 0)
				}

				char, _ := gnomon.GetSCIDValuesByKey(DUELSCID, "ch_b_"+n)
				if char == nil {
					logger.Debugf("[refGetAllDuels] %s no charB\n", n)
					continue
				}

				_, option := gnomon.GetSCIDValuesByKey(DUELSCID, "op_b_"+n)
				if option == nil {
					logger.Debugf("[refGetAllDuels] %s no optB\n", n)
					continue
				}

				_, valA := gnomon.GetSCIDValuesByKey(DUELSCID, "v_a_"+n)
				if valA == nil {
					logger.Debugf("[refGetAllDuels] %s no valA\n", n)
					continue
				}

				_, valB := gnomon.GetSCIDValuesByKey(DUELSCID, "v_b_"+n)
				if valB == nil {
					logger.Debugf("[refGetAllDuels] %s no valB\n", n)
					continue
				}

				var item1Str, item2Str string
				if v.Items >= 1 && v.Opponent.Icon.Item1 == nil {
					item1, _ := gnomon.GetSCIDValuesByKey(DUELSCID, "i1_b_"+n)
					if item1 == nil {
						logger.Debugf("[refGetAllDuels] %s should be a item1\n", n)
						continue
					}

					item1Str = item1[0]
				}

				if v.Items == 2 && v.Opponent.Icon.Item2 == nil {
					item2, _ := gnomon.GetSCIDValuesByKey(DUELSCID, "i2_b_"+n)
					if item2 == nil {
						logger.Debugf("[refGetAllDuels] %s should be a item2\n", n)
						continue
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
				}

				logger.Debugln("[refGetAllDuels] Storing B Info", u)
				Duels.WriteEntry(u, v)
			} else {
				Ready.RemoveIndex(u)
				Duels.Lock()
				delete(Duels.Index, u)
				Duels.Unlock()
			}
		}
	}
}

// Process any Duels that are ready for Ref, will retry any failed ref() txs up to 4 times
func processReady() {
	if !gnomon.IsRunning() {
		return
	}

	for u, e := range Duels.Index {
		if e.Ready > 0 && !e.Complete {
			if !e.validateCollection(true) || !e.validateCollection(false) {
				logger.Warnln("[processReady] Not a valid collection, refunding")
				retry := 0
				tx := Refund(strconv.FormatUint(u, 10))
				time.Sleep(time.Second)
				retry += rpc.ConfirmTxRetry(tx, "refService", 60)
				continue
			}

			now := time.Now().Unix()
			stamp := int64(e.Ready) + 60
			// Will wait a minute after ready stamp before calling refDuel()
			if now > stamp {
				logger.Printf("[processReady] Processing #%s   Death match (%s)   Hardcore (%s)\n", e.Num, e.DM, e.Rule)
				retry := 0
				for retry < 4 {
					tx := e.refDuel()
					time.Sleep(time.Second)
					retry += rpc.ConfirmTxRetry(tx, "refService", 60)
				}
			} else {
				for time.Now().Unix() < stamp {
					logger.Debugf("[processReady] %d waiting for buffer\n", u)
					time.Sleep(time.Second)
				}
				logger.Printf("[processReady] Processing #%s   Death match (%s)   Hardcore (%s)\n", e.Num, e.DM, e.Rule)
				retry := 0
				for retry < 4 {
					tx := e.refDuel()
					time.Sleep(time.Second)
					retry += rpc.ConfirmTxRetry(tx, "refService", 60)
				}
			}
		}
	}
}
