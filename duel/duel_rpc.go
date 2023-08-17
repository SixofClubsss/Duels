package duel

import (
	"fmt"

	"github.com/dReam-dApps/dReams/rpc"
	"github.com/deroproject/derohe/cryptography/crypto"
	dero "github.com/deroproject/derohe/rpc"
)

const DUELSCID = "afde05dd953c0077a9f04dceb2c1b4c58b5153043e67ebef7cf767b63327eac8"

// Start a duel
func StartDuel(amt, items, rule, dm, op uint64, char, item1, item2, token string) {
	rpcClientW, ctx, cancel := rpc.SetWalletClient(rpc.Wallet.Rpc, rpc.Wallet.UserPass)
	defer cancel()

	args := dero.Arguments{
		dero.Argument{Name: "entrypoint", DataType: "S", Value: "Start"},
		dero.Argument{Name: "amt", DataType: "U", Value: amt},
		dero.Argument{Name: "itm", DataType: "U", Value: items},
		dero.Argument{Name: "rule", DataType: "U", Value: rule},
		dero.Argument{Name: "dm", DataType: "U", Value: dm},
		dero.Argument{Name: "op", DataType: "U", Value: op},
		dero.Argument{Name: "ch", DataType: "S", Value: char},
		dero.Argument{Name: "i1", DataType: "S", Value: item1},
		dero.Argument{Name: "i2", DataType: "S", Value: item2},
		dero.Argument{Name: "tkn", DataType: "S", Value: token}}

	var t1 dero.Transfer
	if token != "" {
		t1 = dero.Transfer{
			SCID:   crypto.HashHexToHash(token),
			Amount: 0,
			Burn:   amt,
		}
	} else {
		t1 = dero.Transfer{
			Destination: "dero1qyr8yjnu6cl2c5yqkls0hmxe6rry77kn24nmc5fje6hm9jltyvdd5qq4hn5pn",
			Amount:      0,
			Burn:        amt,
		}
	}

	t2 := dero.Transfer{
		SCID:        crypto.HashHexToHash(char),
		Destination: "dero1qyr8yjnu6cl2c5yqkls0hmxe6rry77kn24nmc5fje6hm9jltyvdd5qq4hn5pn",
		Burn:        1,
	}

	t := []dero.Transfer{t1, t2}
	txid := dero.Transfer_Result{}

	if items == 2 {
		t4 := dero.Transfer{
			SCID:        crypto.HashHexToHash(item2),
			Destination: "dero1qyr8yjnu6cl2c5yqkls0hmxe6rry77kn24nmc5fje6hm9jltyvdd5qq4hn5pn",
			Burn:        1,
		}

		t = append(t, t4)
	}

	if items >= 1 {
		t3 := dero.Transfer{
			SCID:        crypto.HashHexToHash(item1),
			Destination: "dero1qyr8yjnu6cl2c5yqkls0hmxe6rry77kn24nmc5fje6hm9jltyvdd5qq4hn5pn",
			Burn:        1,
		}

		t = append(t, t3)
	}

	fee := rpc.GasEstimate(DUELSCID, "[StartDuel]", args, t, rpc.LowLimitFee)
	params := &dero.Transfer_Params{
		Transfers: t,
		SC_ID:     DUELSCID,
		SC_RPC:    args,
		Ringsize:  2,
		Fees:      fee,
	}

	if err := rpcClientW.CallFor(ctx, &txid, "transfer", params); err != nil {
		logger.Errorln("[StartDuel]", err)
		return
	}

	logger.Println("[StartDuel] Start TX:", txid)
	rpc.AddLog("Start Duel TX: " + txid.TXID)
}

// Accept joinable duel
func (duel entry) AcceptDuel(items, op uint64, char, item1, item2 string) {
	rpcClientW, ctx, cancel := rpc.SetWalletClient(rpc.Wallet.Rpc, rpc.Wallet.UserPass)
	defer cancel()

	args := dero.Arguments{
		dero.Argument{Name: "entrypoint", DataType: "S", Value: "Accept"},
		dero.Argument{Name: "n", DataType: "S", Value: duel.Num},
		dero.Argument{Name: "op", DataType: "U", Value: op},
		dero.Argument{Name: "ch", DataType: "S", Value: char},
		dero.Argument{Name: "i1", DataType: "S", Value: item1},
		dero.Argument{Name: "i2", DataType: "S", Value: item2}}

	var t1 dero.Transfer
	if duel.Token != "" {
		t1 = dero.Transfer{
			SCID:   crypto.HashHexToHash(duel.Token),
			Amount: 0,
			Burn:   duel.Amt,
		}
	} else {
		t1 = dero.Transfer{
			Destination: "dero1qyr8yjnu6cl2c5yqkls0hmxe6rry77kn24nmc5fje6hm9jltyvdd5qq4hn5pn",
			Amount:      0,
			Burn:        duel.Amt,
		}
	}

	t2 := dero.Transfer{
		SCID:        crypto.HashHexToHash(char),
		Destination: "dero1qyr8yjnu6cl2c5yqkls0hmxe6rry77kn24nmc5fje6hm9jltyvdd5qq4hn5pn",
		Burn:        1,
	}

	t := []dero.Transfer{t1, t2}
	txid := dero.Transfer_Result{}

	if items == 2 {
		t4 := dero.Transfer{
			SCID:        crypto.HashHexToHash(item2),
			Destination: "dero1qyr8yjnu6cl2c5yqkls0hmxe6rry77kn24nmc5fje6hm9jltyvdd5qq4hn5pn",
			Burn:        1,
		}

		t = append(t, t4)
	}

	if items >= 1 {
		t3 := dero.Transfer{
			SCID:        crypto.HashHexToHash(item1),
			Destination: "dero1qyr8yjnu6cl2c5yqkls0hmxe6rry77kn24nmc5fje6hm9jltyvdd5qq4hn5pn",
			Burn:        1,
		}

		t = append(t, t3)
	}
	fee := rpc.GasEstimate(DUELSCID, "[AcceptDuel]", args, t, rpc.LowLimitFee)
	params := &dero.Transfer_Params{
		Transfers: t,
		SC_ID:     DUELSCID,
		SC_RPC:    args,
		Ringsize:  2,
		Fees:      fee,
	}

	if err := rpcClientW.CallFor(ctx, &txid, "transfer", params); err != nil {
		logger.Println("[AcceptDuel]", err)
		return
	}

	logger.Println("[AcceptDuel] Accept TX:", txid)
	rpc.AddLog("Accept Duel TX: " + txid.TXID)
}

// Ref a duel, need to be owner or a ref on SC to call
func (duel entry) ref(n, addr string, win rune, odds uint64) (tx string) {
	rpcClientW, ctx, cancel := rpc.SetWalletClient(rpc.Wallet.Rpc, rpc.Wallet.UserPass)
	defer cancel()

	args := dero.Arguments{
		dero.Argument{Name: "entrypoint", DataType: "S", Value: "Ref"},
		dero.Argument{Name: "n", DataType: "S", Value: n},
		dero.Argument{Name: "odds", DataType: "U", Value: odds}}

	t := []dero.Transfer{}
	if duel.DM == "Yes" {
		dst := uint64(0xA1B2C3D4E5F67890)
		var response1, response2 dero.Arguments
		if duel.Items >= 1 {
			var item string
			if win == 'A' {
				item = duel.Opponent.Item1
			} else {
				item = duel.Duelist.Item1
			}

			response1 = dero.Arguments{
				{Name: dero.RPC_DESTINATION_PORT, DataType: dero.DataUint64, Value: dst},
				{Name: dero.RPC_SOURCE_PORT, DataType: dero.DataUint64, Value: uint64(0)},
				{Name: dero.RPC_COMMENT, DataType: dero.DataString, Value: fmt.Sprintf("You've won  %s  in a Duel Death Match", item)}}

			t1 := dero.Transfer{
				Destination: addr,
				Amount:      1,
				Payload_RPC: response1}

			t = append(t, t1)
		}

		if duel.Items == 2 {
			var item string
			if win == 'A' {
				item = duel.Opponent.Item2
			} else {
				item = duel.Duelist.Item2
			}

			response2 = dero.Arguments{
				{Name: dero.RPC_DESTINATION_PORT, DataType: dero.DataUint64, Value: dst},
				{Name: dero.RPC_SOURCE_PORT, DataType: dero.DataUint64, Value: uint64(0)},
				{Name: dero.RPC_COMMENT, DataType: dero.DataString, Value: fmt.Sprintf("You've won  %s  in a Duel Death Match", item)}}

			t2 := dero.Transfer{
				Destination: addr,
				Amount:      1,
				Payload_RPC: response2}

			t = append(t, t2)
		}
	}

	txid := dero.Transfer_Result{}
	fee := rpc.GasEstimate(DUELSCID, "[refDuel]", args, t, rpc.LowLimitFee)
	params := &dero.Transfer_Params{
		Transfers: t,
		SC_ID:     DUELSCID,
		SC_RPC:    args,
		Ringsize:  2,
		Fees:      fee,
	}

	if err := rpcClientW.CallFor(ctx, &txid, "transfer", params); err != nil {
		logger.Errorln("[refDuel]", err)
		return
	}

	logger.Println("[refDuel] Ref Duel TX:", txid)
	rpc.AddLog("Ref Duel TX: " + txid.TXID)

	return txid.TXID
}

// Revive a character from graveyard
func (grave grave) Revive() {
	rpcClientW, ctx, cancel := rpc.SetWalletClient(rpc.Wallet.Rpc, rpc.Wallet.UserPass)
	defer cancel()

	args := dero.Arguments{
		dero.Argument{Name: "entrypoint", DataType: "S", Value: "Revi"},
		dero.Argument{Name: "n", DataType: "S", Value: grave.Num},
		dero.Argument{Name: "asset", DataType: "S", Value: grave.Char},
	}

	var t1 dero.Transfer
	if grave.Token != "" {
		t1 = dero.Transfer{
			SCID:   crypto.HashHexToHash(grave.Token),
			Amount: 0,
			Burn:   grave.findDiscount(),
		}
	} else {
		t1 = dero.Transfer{
			Destination: "dero1qyr8yjnu6cl2c5yqkls0hmxe6rry77kn24nmc5fje6hm9jltyvdd5qq4hn5pn",
			Amount:      0,
			Burn:        grave.findDiscount(),
		}
	}

	t := []dero.Transfer{t1}
	txid := dero.Transfer_Result{}
	fee := rpc.GasEstimate(DUELSCID, "[Revive]", args, t, rpc.LowLimitFee)
	params := &dero.Transfer_Params{
		Transfers: t,
		SC_ID:     DUELSCID,
		SC_RPC:    args,
		Ringsize:  2,
		Fees:      fee,
	}

	if err := rpcClientW.CallFor(ctx, &txid, "transfer", params); err != nil {
		logger.Errorln("[Revive]", err)
		return
	}

	logger.Println("[Revive] Revive TX:", txid)
	rpc.AddLog("Revive TX: " + txid.TXID)
}

// Refund a duel, used by owners, refs and players
func Refund(n string) {
	rpcClientW, ctx, cancel := rpc.SetWalletClient(rpc.Wallet.Rpc, rpc.Wallet.UserPass)
	defer cancel()
	tag := "Refund"

	args := dero.Arguments{
		dero.Argument{Name: "entrypoint", DataType: "S", Value: "Refund"},
		dero.Argument{Name: "n", DataType: "S", Value: n},
	}

	t := []dero.Transfer{}
	txid := dero.Transfer_Result{}
	fee := rpc.GasEstimate(DUELSCID, fmt.Sprintf("[%s]", tag), args, t, rpc.LowLimitFee)
	params := &dero.Transfer_Params{
		Transfers: t,
		SC_ID:     DUELSCID,
		SC_RPC:    args,
		Ringsize:  2,
		Fees:      fee,
	}

	if err := rpcClientW.CallFor(ctx, &txid, "transfer", params); err != nil {
		logger.Errorf("[%s] %s", tag, err)
		return
	}

	logger.Printf("[Refund] Refund TX: %s\n", txid)
	rpc.AddLog("Refund TX: " + txid.TXID)
}
