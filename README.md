# Duels
Dero asset duels.

Written in Go and using [Fyne Toolkit](https://fyne.io/), **Duels** is built on Dero's private L1. Powered by [Gnomon](https://github.com/civilware/Gnomon) and [dReams](https://github.com/dReam-dApps/dReams), **Duels** allows Dero users to pit their Dero assets against each other in PvP duels. Modeled after the duels that took place in the days of wild west, **Duels** has a variety of game modes to choose from. The game mechanics are essentially higher/lower, where the outcome is derived on chain.


![goMod](https://img.shields.io/github/go-mod/go-version/SixofClubsss/Duels.svg)![goReport](https://goreportcard.com/badge/github.com/SixofClubsss/Duels)[![goDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/SixofClubsss/Duels)

Duels dApp with full features is available for download from [dReams](https://dreamdapps.io).

![windowsOS](https://raw.githubusercontent.com/SixofClubsss/dreamdappsite/main/assets/os-windows-green.svg)![macOS](https://raw.githubusercontent.com/SixofClubsss/dreamdappsite/main/assets/os-macOS-green.svg)![linuxOS](https://raw.githubusercontent.com/SixofClubsss/dreamdappsite/main/assets/os-linux-green.svg)

### Features
- Regular Duels (odds used, requires ref)
- Death match Duels (winner gets defeated characters items)
- Hardcore Duels (no odds or ref, quick pace)
- Combine hardcore and death match modes
- Graveyard with defeated character revivals
- Outcome determined by SC
- Character and item assets with ranking system
- Payout odds determined by ranking system
- Leader board to track wins/loses
- Multiple currencies supported
- Auto claim NFA assets
- Local DB storage
- Ref service

### Build
Following these build instructions, you can build Duels as a *individual* dApp.
- Install latest [Go version](https://go.dev/doc/install)
- Install [Fyne](https://developer.fyne.io/started/) dependencies
- Clone repo and build using:
```
git clone https://github.com/SixofClubsss/Duels.git
cd Duels/cmd/Duels
go build .
./Duels
```

### Using service
Ref service allows for contract owners to automatically process any Duels that require a referee. Up to 9 refs can be added to a contract. 
- Install latest [Go version](https://go.dev/doc/install)
- Clone repo and build with:

```
git clone https://github.com/SixofClubsss/Duels.git
cd Duels/cmd/refService
go build .
```
- Options
```
Options:
  -h --help                      Show this screen.
  --daemon=<127.0.0.1:10102>     Set daemon rpc address to connect.
  --wallet=<127.0.0.1:10103>     Set wallet rpc address to connect.
  --login=<user:pass>     	 Wallet rpc user:pass for auth.
  --fastsync=<true>	         Gnomon option,  true/false value to define loading at chain height on start up.
  --num-parallel-blocks=<5>      Gnomon option,  defines the number of parallel blocks to index.`

```

- On local daemon, with wallet running rpc server start the service using:
```
./refService --login=user:pass
```

### Donations
- **Dero Address**: dero1qyr8yjnu6cl2c5yqkls0hmxe6rry77kn24nmc5fje6hm9jltyvdd5qq4hn5pn

![DeroDonations](https://raw.githubusercontent.com/SixofClubsss/dreamdappsite/main/assets/DeroDonations.jpg)

---

#### Licensing

Duels is free and open source.   
The source code is published under the [MIT](https://github.com/SixofClubsss/Duels/blob/main/LICENSE) License.   
Copyright © 2023 SixofClubs   
