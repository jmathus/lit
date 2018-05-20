package qln

import (
	"fmt"
  "time"
  "crypto/sha1"
  "math/rand"

	"github.com/mit-dci/lit/lnutil"
)

// ExchangeChannels initiates a state update by setting up HTLC's
// TODO.jesus
func (nd LitNode) AssignHTLC(qc *Qchan, amt uint32, htlc *HTLC, incoming bool) error {
	// sanity checks
	if amt >= 1<<30 {
		return fmt.Errorf("max send 1G sat (1073741823)")
	}
	if amt == 0 {
		return fmt.Errorf("have to send non-zero amount")
	}
  if amt < 0 {
    return fmt.Errorf("amount must be denoted as positive value")
  }

	// see if channel is busy, error if so, lock if not
	// lock this channel

	select {
	case <-qc.ClearToSend:
	// keep going
	default:
		return fmt.Errorf("Channel %d busy", qc.Idx())
	}
	// ClearToSend is now empty

	// reload from disk here, after unlock
	err := nd.ReloadQchanState(qc)
	if err != nil {
		// don't clear to send here; something is wrong with the channel
		return err
	}

	// check that channel is confirmed, if non-test coin
	wal, ok := nd.SubWallet[qc.Coin()]
	if !ok {
		qc.ClearToSend <- true
		return fmt.Errorf("Not connected to coin type %d\n", qc.Coin())
	}

	if !wal.Params().TestCoin && qc.Height < 100 {
		qc.ClearToSend <- true
		return fmt.Errorf(
			"height %d; must wait min 1 conf for non-test coin\n", qc.Height)
	}

	// TODO.jesus Assign the HTLC to the State
	qc.State.CurrentHTLC = htlc

	// Check that the amount passed is this method equals the amount passed in the htlc
	if (int64(amt) != htlc.ExchangeAmount) {
		return fmt.Errorf("Mismatch between given amount and amount found in given HTLC")
	}

	// perform minOutput checks after reload
	myNewOutputSize := (qc.State.MyAmt - int64(amt)) - qc.State.Fee
	theirNewOutputSize := qc.Value - (qc.State.MyAmt - int64(amt)) - qc.State.Fee
	// If incoming, overwrite with correct amounts
	if (incoming) {
		myNewOutputSize = qc.State.MyAmt + int64(amt) + qc.State.Fee
		theirNewOutputSize = qc.Value - (qc.State.MyAmt - int64(amt)) - qc.State.Fee
	}

	// check if this push would lower my balance below minBal
	if myNewOutputSize < minOutput {
		qc.ClearToSend <- true
		return fmt.Errorf("want to push %s but %s available, %s fee, %s minOutput",
			lnutil.SatoshiColor(int64(amt)),
			lnutil.SatoshiColor(qc.State.MyAmt),
			lnutil.SatoshiColor(qc.State.Fee),
			lnutil.SatoshiColor(minOutput))
	}
	// check if this push is sufficient to get them above minBal
	if theirNewOutputSize < minOutput {
		qc.ClearToSend <- true
		return fmt.Errorf(
			"pushing %s insufficient; counterparty bal %s fee %s minOutput %s",
			lnutil.SatoshiColor(int64(amt)),
			lnutil.SatoshiColor(qc.Value-qc.State.MyAmt),
			lnutil.SatoshiColor(qc.State.Fee),
			lnutil.SatoshiColor(minOutput))
	}

	// if we got here, but channel is not in rest state, try to fix it.
	// TODO.jesus?AddCheck How to resend msg
	// if qc.State.Delta != 0 {
	// 	err = nd.AssignHTLCReSendMsg(qc, incoming)
	// 	if err != nil {
	// 		qc.ClearToSend <- true
	// 		return err
	// 	}
	// 	qc.ClearToSend <- true
	// 	return fmt.Errorf("Didn't send.  Recovered though, so try again!")
	// }

  if (incoming) {
		qc.State.Delta = int32(amt)
	} else {
		qc.State.Delta = int32(-amt)
	}

	// save to db with ONLY htlc and delta changed
	err = nd.SaveQchanState(qc)
	if err != nil {
		// don't clear to send here; something is wrong with the channel
		return err
	}

	err = nd.AssignHTLCSendDeltaSig(qc, incoming, htlc)
	if err != nil {
		// don't clear; something is wrong with the network
		return err
	}

	fmt.Printf("got pre CTS... \n")
	// block until clear to send is full again
	<-qc.ClearToSend
	fmt.Printf("got post CTS... \n")
	// since we cleared with that statement, fill it again before returning
	qc.ClearToSend <- true

	return nil
}

func (nd LitNode) OpenHTLC(qc *Qchan, amt uint32, preimage []int32, incoming bool) error {
  // Checks the preimage to unlock the HTLC
  hash := sha1.New()
  hash.Write([]byte(string(preimage)))
  preRHash := hash.Sum(nil)
  var computedRHash [20]byte
  copy(computedRHash[:], preRHash)

  if (computedRHash != qc.State.CurrentHTLC.RHash) {
    return fmt.Errorf("Wrong preimage, cannot unlock this HTLC")
  }

  // Checks that the HTLC is opened within the given time limit, default of 1 minute right now
  currentTime := time.Now().UTC()

  if (currentTime.After(qc.State.CurrentHTLC.Locktime)) {
    return fmt.Errorf("Time limit reached, HTLC expired at %s", qc.State.CurrentHTLC.Locktime)
  }

	// see if channel is busy, error if so, lock if not
	// lock this channel
	select {
	case <-qc.ClearToSend:
	// keep going
	default:
		return fmt.Errorf("Channel %d busy", qc.Idx())
	}
	// ClearToSend is now empty

	// reload from disk here, after unlock
	err := nd.ReloadQchanState(qc)
	if err != nil {
		// don't clear to send here; something is wrong with the channel
		return err
	}

	// check that channel is confirmed, if non-test coin
	wal, ok := nd.SubWallet[qc.Coin()]
	if !ok {
		qc.ClearToSend <- true
		return fmt.Errorf("Not connected to coin type %d\n", qc.Coin())
	}

	if !wal.Params().TestCoin && qc.Height < 100 {
		qc.ClearToSend <- true
		return fmt.Errorf(
			"height %d; must wait min 1 conf for non-test coin\n", qc.Height)
	}

	// Check that the amount in the HTLC being manipulated is equal to the amount that we
	// are currently trying to exchange (to make sure there is no HTLC mixup)
	if (int64(amt) != qc.State.CurrentHTLC.ExchangeAmount) {
		return fmt.Errorf("Wrong HTLC: amount in HTLC is %d, but are exchanging %d", qc.State.CurrentHTLC.ExchangeAmount, amt)
	}

	// TODO.jesus?AddCheck How to resend msg
	// if we got here, but channel is not in rest state, try to fix it.
	// if qc.State.Delta != 0 {
	// 	err = nd.OpenHTLCReSendMsg(qc)
	// 	if err != nil {
	// 		qc.ClearToSend <- true
	// 		return err
	// 	}
	// 	qc.ClearToSend <- true
	// 	return fmt.Errorf("Didn't send.  Recovered though, so try again!")
	// }

	if (incoming) {
		qc.State.Delta = int32(qc.State.CurrentHTLC.ExchangeAmount)
	} else {
		qc.State.Delta = int32(-qc.State.CurrentHTLC.ExchangeAmount)
	}

	// save to db with ONLY delta changed
	err = nd.SaveQchanState(qc)
	if err != nil {
		// don't clear to send here; something is wrong with the channel
		return err
	}
	// move unlock to here so that delta is saved before

	err = nd.OpenHTLCSendDeltaSig(qc, incoming, preimage)
	if err != nil {
		// don't clear; something is wrong with the network
		return err
	}

	fmt.Printf("got pre CTS... \n")
	// block until clear to send is full again
	<-qc.ClearToSend
	fmt.Printf("got post CTS... \n")
	// since we cleared with that statement, fill it again before returning
	qc.ClearToSend <- true

	return nil
}

func (nd *LitNode) SendExchangeRequest(qc1 *Qchan, qc2 *Qchan, amt1 int64, amt2 int64) error {
	outMsg := lnutil.NewExchangeRequestMsg(qc1.Peer(), qc1.Op, qc1.Peer(), amt1, qc2.Peer(), amt2)
	nd.OmniOut <- outMsg

	return nil
}

func (nd *LitNode) ExchangeRequestHandler(msg lnutil.ExchangeRequestMsg, qc *Qchan) error {
  // Generate a requestID and expiration time (default will be a 1 minute expiration time for requests)
  var characters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	rand.Seed(time.Now().UTC().UnixNano())
	var emptyTime time.Time

	runeArr := make([]rune, 20)
	for i := range runeArr {
			runeArr[i] = characters[rand.Intn(len(characters))]
	}
	requestID := string(runeArr)
  expirationTime := time.Now().UTC().Add(time.Minute*1)

  request := new(Request)
  request.ChanIdx1 = msg.ChanIdx1
  request.Amt1 = msg.Amt1
  request.ChanIdx2 = msg.ChanIdx2
  request.Amt2 = msg.Amt2
  request.ExpirationTime = expirationTime
  request.RequestID = requestID

  qc.State.CurrentRequest = request
	nd.CurrentRequest = request

  // Print the request message for the peer to decide to accept or not
  fmt.Printf("Exchange request: %s to you on channel %s for %s from you on channel %s\n", msg.Amt1, msg.ChanIdx1, msg.Amt2, msg.ChanIdx2)
  fmt.Printf("RequestID: %s Expiration Time (UTC): %s\n", requestID, expirationTime.Format("2006-01-02 15:04:05"))
  fmt.Printf("To accept, use command: respond <YES> <requestID>")
  fmt.Printf("To decline, use command: respond <NO> <requestID>")

  for {
    if (expirationTime.Before(time.Now().UTC())) {
      qc.State.CurrentRequest = new(Request)
			nd.CurrentRequest = new(Request)
      break
    }
		// This is reached if the user accepts or declines the exchange before the expirationTime is reached,
		// and is put in place to avoid future request creation from being overwritten by this loop
		if (qc.State.CurrentRequest.ExpirationTime.Equal(emptyTime)) {
			break
		}
  }

	return nil
}
