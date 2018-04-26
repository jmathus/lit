package main

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"unicode"

	"github.com/fatih/color"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
)

var fundCommand = &Command{
	Format: fmt.Sprintf("%s%s%s\n", lnutil.White("fund"),
		lnutil.ReqColor("peer", "coinType", "capacity", "initialSend"), lnutil.OptColor("data")),
	Description: fmt.Sprintf("%s\n%s\n%s\n%s\n",
		"Establish and fund a new lightning channel with the given peer.",
		"The capacity is the amount of satoshi we insert into the channel,",
		"and initialSend is the amount we initially hand over to the other party.",
		"data is an optional field that can contain 32 bytes of hex to send as part of the channel fund",
	),
	ShortDescription: "Establish and fund a new lightning channel with the given peer.\n",
}

var pushCommand = &Command{
	Format: fmt.Sprintf("%s%s%s%s\n", lnutil.White("push"), lnutil.ReqColor("channel idx", "amount"), lnutil.OptColor("times"), lnutil.OptColor("data")),
	Description: fmt.Sprintf("%s\n%s\n%s\n",
		"Push the given amount (in satoshis) to the other party on the given channel.",
		"Optionally, the push operation can be associated with a 32 byte value hex encoded.",
		"Optionally, the push operation can be repeated <times> number of times."),
	ShortDescription: "Push the given amount (in satoshis) to the other party on the given channel.\n",
}

var closeCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("close"), lnutil.ReqColor("channel idx")),
	Description: fmt.Sprintf("%s\n%s\n%s%s\n",
		"Cooperatively close the channel with the given index by asking",
		"the other party to finalize the channel pay-out.",
		"See also: ", lnutil.White("break")),
	ShortDescription: "Cooperatively close the channel with the given index by asking\n",
}

var breakCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("break"), lnutil.ReqColor("channel idx")),
	Description: fmt.Sprintf("%s\n%s\n%s%s\n",
		"Forcibly break the given channel. Note that you need to wait",
		"a set number of blocks before you can use the money.",
		"See also: ", lnutil.White("stop")),
	ShortDescription: "Forcibly break the given channel.\n",
}

var historyCommand = &Command{
	Format:           lnutil.White("history"),
	Description:      "Show all the metadata for justice txs",
	ShortDescription: "Show all the metadata for justice txs.\n",
}

// TODO.jesus
var exchangeCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("exchange"), lnutil.ReqColor("channel idx one", "amount sent", "channel idx two", "amount requested")),
	Description: fmt.Sprintf("%s\n%s\n",
		"Exchange the amounts proposed on the two different channels listed (with each channel ideally having a different currency)",
		"Optionally, the exchange operation can be accepted by the receiver to finalize the exchange through HTLC mechanisms"),
	ShortDescription: "Send an exchange proposal for the given amounts (in different currencies) to the other party on the two given channels.\n",
}

// TODO.jesus
var respondCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("respond"), lnutil.ReqColor("YES/NO", "request id")),
	Description: fmt.Sprintf("%s\n", "Accepts or declines the corresponding exchange request as specified by the requestID"),
	ShortDescription: "Sends a response to the corresponding exchange request\n",
}

// TODO.jesus
var priceCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("price"), lnutil.ReqColor("cryptocurrency name")),
	Description: fmt.Sprintf("%s\n", "Returns the current market price of the desired cryptocurrency"),
	ShortDescription: "Returns the current market price of the desired cryptocurrency\n",
}

// TODO.jesus
var compareCommand = &Command{
	Format: fmt.Sprintf("%s%s\n", lnutil.White("compare"), lnutil.ReqColor("first cryptocurrency name", "second cryptocurrency name")),
	Description: fmt.Sprintf("%s\n", "Returns the price of the second currency compared to the first"),
	ShortDescription: "Returns the price of the second currency in relation to one unit of the first currency\n",
}

func (lc *litAfClient) History(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, historyCommand.Format)
		fmt.Fprintf(color.Output, historyCommand.Description)
		return nil
	}

	args := new(litrpc.StateDumpArgs)
	reply := new(litrpc.StateDumpReply)

	err := lc.rpccon.Call("LitRPC.StateDump", args, reply)
	if err != nil {
		return err
	}

	for _, tx := range reply.Txs {
		fmt.Fprintf(color.Output, "Pkh: %x, Idx: %d, Sig: %x, Txid: %x, Data: %x, Amt: %d\n", tx.Pkh, tx.Idx, tx.Sig, tx.Txid, tx.Data, tx.Amt)
	}

	return nil
}

func (lc *litAfClient) FundChannel(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, fundCommand.Format)
		fmt.Fprintf(color.Output, fundCommand.Description)
		return nil
	}

	args := new(litrpc.FundArgs)
	reply := new(litrpc.StatusReply)

	if len(textArgs) < 4 {
		return fmt.Errorf(fundCommand.Format)
	}

	peer, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}
	coinType, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}

	cCap, err := strconv.Atoi(textArgs[2])
	if err != nil {
		return err
	}
	iSend, err := strconv.Atoi(textArgs[3])
	if err != nil {
		return err
	}

	if len(textArgs) > 4 {
		data, err := hex.DecodeString(textArgs[4])
		if err != nil {
			// Wasn't valid hex, copy directly and truncate
			copy(args.Data[:], textArgs[3])
		} else {
			copy(args.Data[:], data[:])
		}
	}

	args.Peer = uint32(peer)
	args.CoinType = uint32(coinType)
	args.Capacity = int64(cCap)
	args.InitialSend = int64(iSend)

	err = lc.rpccon.Call("LitRPC.FundChannel", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}

// Request close of a channel.  Need to pass in peer, channel index
func (lc *litAfClient) CloseChannel(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, closeCommand.Format)
		fmt.Fprintf(color.Output, closeCommand.Description)
		return nil
	}

	args := new(litrpc.ChanArgs)
	reply := new(litrpc.StatusReply)

	// need args, fail
	if len(textArgs) < 1 {
		return fmt.Errorf("need args: close chanIdx")
	}

	cIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}

	args.ChanIdx = uint32(cIdx)

	err = lc.rpccon.Call("LitRPC.CloseChannel", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}

// Almost exactly the same as CloseChannel.  Maybe make "break" a bool...?
func (lc *litAfClient) BreakChannel(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, breakCommand.Format)
		fmt.Fprintf(color.Output, breakCommand.Description)
		return nil
	}

	args := new(litrpc.ChanArgs)
	reply := new(litrpc.StatusReply)

	// need args, fail
	if len(textArgs) < 1 {
		return fmt.Errorf("need args: break chanIdx")
	}

	cIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}

	args.ChanIdx = uint32(cIdx)

	err = lc.rpccon.Call("LitRPC.BreakChannel", args, reply)
	if err != nil {
		return err
	}

	fmt.Fprintf(color.Output, "%s\n", reply.Status)
	return nil
}

// Push is the shell command which calls PushChannel
func (lc *litAfClient) Push(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, pushCommand.Format)
		fmt.Fprintf(color.Output, pushCommand.Description)
		return nil
	}

	args := new(litrpc.PushArgs)
	reply := new(litrpc.PushReply)

	if len(textArgs) < 2 {
		return fmt.Errorf("need args: push chanIdx amt (times) (data)")
	}

	// this stuff is all the same as in cclose, should put into a function...
	cIdx, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}
	amt, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}

	times := int(1)
	if len(textArgs) > 2 {
		times, err = strconv.Atoi(textArgs[2])
		if err != nil {
			return err
		}
	}

	if len(textArgs) > 3 {
		data, err := hex.DecodeString(textArgs[3])
		if err != nil {
			// Wasn't valid hex, copy directly and truncate
			copy(args.Data[:], textArgs[3])
		} else {
			copy(args.Data[:], data[:])
		}
	}

	args.ChanIdx = uint32(cIdx)
	args.Amt = int64(amt)

	for times > 0 {
		err := lc.rpccon.Call("LitRPC.Push", args, reply)
		if err != nil {
			return err
		}
		fmt.Fprintf(color.Output, "Pushed %s at state %s\n", lnutil.SatoshiColor(int64(amt)), lnutil.White(reply.StateIndex))
		times--
	}

	return nil
}

func (lc *litAfClient) Dump(textArgs []string) error {
	pReply := new(litrpc.DumpReply)
	pArgs := new(litrpc.NoArgs)

	err := lc.rpccon.Call("LitRPC.DumpPrivs", pArgs, pReply)
	if err != nil {
		return err
	}
	fmt.Fprintf(color.Output, "Private keys for all channels and utxos:\n")

	// Display DumpPriv info
	for i, t := range pReply.Privs {
		fmt.Fprintf(color.Output, "%d %s h:%d amt:%s %s ",
			i, lnutil.OutPoint(t.OutPoint), t.Height,
			lnutil.SatoshiColor(t.Amt), t.CoinType)
		if t.Delay != 0 {
			fmt.Fprintf(color.Output, " delay: %d", t.Delay)
		}
		if !t.Witty {
			fmt.Fprintf(color.Output, " non-witness")
		}
		if len(t.PairKey) > 1 {
			fmt.Fprintf(
				color.Output, "\nPair Pubkey: %s", lnutil.Green(t.PairKey))
		}
		fmt.Fprintf(color.Output, "\n\tprivkey: %s", lnutil.Red(t.WIF))
		fmt.Fprintf(color.Output, "\n")
	}

	return nil
}

// TODO.jesus
// Exchange is the shell command which calls exchangeRequest
// Do I need more args like (time) or (data)???
func (lc *litAfClient) Exchange(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, exchangeCommand.Format)
		fmt.Fprintf(color.Output, exchangeCommand.Description)
		return nil
	}

	args := new(litrpc.ExchangeArgs)
	reply := new(litrpc.ExchangeReply)

	if len(textArgs) < 4 {
		return fmt.Errorf("need args: exchange chanIdx1 amt1 chanIdx2 amt2 (times) (data)")
	}

	chanId1, err := strconv.Atoi(textArgs[0])
	if err != nil {
		return err
	}
	amount1, err := strconv.Atoi(textArgs[1])
	if err != nil {
		return err
	}
	chanId2, err := strconv.Atoi(textArgs[2])
	if err != nil {
		return err
	}
	amount2, err := strconv.Atoi(textArgs[3])
	if err != nil {
		return err
	}


	times := int(1)
	if len(textArgs) > 4 {
		times, err = strconv.Atoi(textArgs[4])
		if err != nil {
			return err
		}
	}

	if len(textArgs) > 5 {
		data, err := hex.DecodeString(textArgs[5])
		if err != nil {
			// Wasn't valid hex, copy directly and truncate
			copy(args.Data[:], textArgs[5])
		} else {
			copy(args.Data[:], data[:])
		}
	}

	args.ChanIdx1 = uint32(chanId1)
	args.Amt1 = int64(amount1)
	args.ChanIdx2 = uint32(chanId2)
	args.Amt2 = int64(amount2)

	for times > 0 {
		err := lc.rpccon.Call("LitRPC.Exchange", args, reply)
		if err != nil {
			return err
		}
		// TODO.jesus Edit to add currencies used
		fmt.Fprintf(color.Output, "Exchanged %s at state %s for %s\n", lnutil.SatoshiColor(int64(amount1)), lnutil.White(reply.StateIndex), lnutil.SatoshiColor(int64(amount2)))
		times--
	}

	return nil
}

// TODO.jesus
// A shell command that responds to an exchange request either accepting or declining
func (lc *litAfClient) Respond(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, respondCommand.Format)
		fmt.Fprintf(color.Output, respondCommand.Description)
		return nil
	}

	if len(textArgs) < 2 {
		return fmt.Errorf("need args: respond YES/NO requestID")
	}

	args := new(litrpc.RespondArgs)
	reply := new(litrpc.RespondReply)

	args.Yesno = textArgs[0]
	args.RequestID = textArgs[1]

	times := int(1)
	for times > 0 {
		err := lc.rpccon.Call("LitRPC.Respond", args, reply)
		if err != nil {
			return err
		}
		// TODO.jesus Edit to add currencies used
		fmt.Fprintf(color.Output, "Respond %s to exchange request: %s at state %s\n", lnutil.White(textArgs[0]), lnutil.White(textArgs[1]), lnutil.White(reply.StateIndex))
		times--
	}

	return nil
}

// TODO.jesus
// A shell command that returns the price of the desired cryptocurrency
func (lc *litAfClient) Price(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, priceCommand.Format)
		fmt.Fprintf(color.Output, priceCommand.Description)
		return nil
	}

	if len(textArgs) < 1 {
		return fmt.Errorf("need args: price currencyName")
	}

	runeArr := []rune(textArgs[0])
  currencyName := ""

  for i := range runeArr {
    if i == 0 {
      currencyName += string(unicode.ToUpper(runeArr[i]))
    } else{
      currencyName += string(unicode.ToLower(runeArr[i]))
    }
  }

	fmt.Printf("%s: %s\n", currencyName, getCryptocurrencyPrice(currencyName))

	return nil
}

// TODO.jesus
// A shell command that returns the price of the desired first cryptocurrency compared to the second
func (lc *litAfClient) Compare(textArgs []string) error {
	if len(textArgs) > 0 && textArgs[0] == "-h" {
		fmt.Fprintf(color.Output, compareCommand.Format)
		fmt.Fprintf(color.Output, compareCommand.Description)
		return nil
	}

	if len(textArgs) < 2 {
		return fmt.Errorf("need args: compare currencyName1 currencyName2")
	}

	runeArr1 := []rune(textArgs[0])
  currencyName1 := ""

  for i := range runeArr1 {
    if i == 0 {
      currencyName1 += string(unicode.ToUpper(runeArr1[i]))
    } else{
      currencyName1 += string(unicode.ToLower(runeArr1[i]))
    }
  }

	runeArr2 := []rune(textArgs[1])
  currencyName2 := ""

  for i := range runeArr2 {
    if i == 0 {
      currencyName2 += string(unicode.ToUpper(runeArr2[i]))
    } else{
      currencyName2 += string(unicode.ToLower(runeArr2[i]))
    }
  }

	fmt.Println(getCryptocurrencyExchangeRate(currencyName1, currencyName2))

	return nil
}
