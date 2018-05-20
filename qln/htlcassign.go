package qln

import (
	"fmt"

	//"github.com/adiabat/btcd/wire"
	"github.com/mit-dci/lit/lnutil"
)

// SendNextMsg determines what message needs to be sent next
// based on the channel state.  It then calls the appropriate function.
// func (nd *LitNode) AssignHTLCReSendMsg(qc *Qchan, incoming bool) error {
//   if (incoming) {
//     // DeltaSig
//   	if qc.State.Delta > 0 {
//   		fmt.Printf("Sending previously sent AssignHTLCDeltaSig\n")
//   		return nd.AssignHTLCSendDeltaSig(qc)
//   	}
//
//   	// SigRev
//   	if qc.State.Delta < 0 {
//   		fmt.Printf("Sending previously sent AssignHTLCSigRev\n")
//   		return nd.AssignHTLCSendSigRev(qc)
//   	}
//
//   	// Rev
//   	return nd.AssignHTLCSendREV(qc)
//   } else {
//     // DeltaSig
//   	if qc.State.Delta < 0 {
//   		fmt.Printf("Sending previously sent AssignHTLCDeltaSig\n")
//   		return nd.AssignHTLCSendDeltaSig(qc)
//   	}
//
//   	// SigRev
//   	if qc.State.Delta > 0 {
//   		fmt.Printf("Sending previously sent AssignHTLCSigRev\n")
//   		return nd.AssignHTLCSendSigRev(qc)
//   	}
//
//   	// Rev
//   	return nd.AssignHTLCSendREV(qc)
//   }
// }

func (nd *LitNode) AssignHTLCSendDeltaSig(q *Qchan, incoming bool, htlc *HTLC) error {
	// increment state number, update balance, go to next elkpoint
	q.State.StateIdx++
  if (!incoming) {
    q.State.MyAmt += int64(q.State.Delta)
  }
	q.State.ElkPoint = q.State.NextElkPoint
	q.State.NextElkPoint = q.State.N2ElkPoint
	// N2Elk is now invalid

	// make the signature to send over
	sig, err := nd.SignState(q)
	if err != nil {
		return err
	}

	peerHTLC := new(lnutil.HTLC)
	peerHTLC.Incoming = !htlc.Incoming
	peerHTLC.Qchan1 = htlc.Qchan1
	peerHTLC.ExchangeAmount = htlc.ExchangeAmount
	peerHTLC.RHash = htlc.RHash
	peerHTLC.Locktime = htlc.Locktime

  outMsg := lnutil.NewAssignHTLCDeltaSigMsg(q.Peer(), q.Op, -q.State.Delta, !incoming, *peerHTLC, sig, q.State.Data)
	nd.OmniOut <- outMsg

	return nil
}

func (nd *LitNode) AssignHTLCDeltaSigHandler(msg lnutil.AssignHTLCDeltaSigMsg, qc *Qchan) error {

	var collision bool
	//incomingDelta := uint32(math.Abs(float64(msg.Delta))) //int32 (may be negative, but should not be)
	incomingDelta := msg.Delta

	// we should be clear to send when we get a AssignHTLCDeltaSig
	select {
	case <-qc.ClearToSend:
	// keep going, normal
	default:
		// collision
		collision = true
	}

	fmt.Printf("COLLISION is (%s)\n", collision)

	// load state from disk
	err := nd.ReloadQchanState(qc)
	if err != nil {
		return fmt.Errorf("AssignHTLCDeltaSigHandler ReloadQchan err %s", err.Error())
	}

	// TODO we should send a response that the channel is closed.
	// or offer to double spend with a cooperative close?
	// or update the remote node on closed channel status when connecting
	// TODO should disallow 'break' command when connected to the other node
	// or merge 'break' and 'close' UI so that it breaks when it can't
	// connect, and closes when it can.
	if qc.CloseData.Closed {
		return fmt.Errorf("AssignHTLCDeltaSigHandler err: %d, %d is closed.",
			qc.Peer(), qc.Idx())
	}

	if collision {
		// incoming delta saved as collision value,
		// existing (negative) delta value retained.
    if (msg.Incoming) {
      qc.State.Collision = int32(0)
    } else {
      qc.State.Collision = int32(incomingDelta)
    }
		fmt.Printf("AssignHTLC delta sig COLLISION (%d)\n", qc.State.Collision)
	}

	// detect if channel is already locked, and lock if not
	//	nd.PushClearMutex.Lock()
	//	if nd.PushClear[qc.Idx()] == nil {
	//		nd.PushClear[qc.Idx()] = make(chan bool, 1)
	//	} else {
	// this means there was a collision
	// reload from disk; collision may have happened after disk read
	//		err := nd.ReloadQchan(qc)
	//		if err != nil {
	//			return fmt.Errorf("DeltaSigHandler err %s", err.Error())
	//		}

	//	}

  // TODO.jesus?AddCheck
  // if (msg.Incoming) {
  //   if qc.State.Delta < 0 {
  // 		fmt.Printf(
  // 			"DeltaSigHandler err: chan %d delta %d, expect rev, send empty rev",
  // 			qc.Idx(), qc.State.Delta)
  //
  // 		return nd.AssignHTLCSendREV(qc)
  // 	}
  // } else {
  //   if qc.State.Delta > 0 {
  // 		fmt.Printf(
  // 			"DeltaSigHandler err: chan %d delta %d, expect rev, send empty rev",
  // 			qc.Idx(), qc.State.Delta)
  //
  // 		return nd.AssignHTLCSendREV(qc)
  // 	}
  // }

	if !collision {
		// no collision, incoming (negative) delta or zero saved based on funds going to HTLC from here or not
    qc.State.Delta = int32(incomingDelta)

		copyHTLC := new(HTLC)
		copyHTLC.Incoming = msg.HTLC.Incoming
		copyHTLC.Qchan1 = msg.HTLC.Qchan1
		copyHTLC.ExchangeAmount = msg.HTLC.ExchangeAmount
		copyHTLC.RHash = msg.HTLC.RHash
		copyHTLC.Locktime = msg.HTLC.Locktime


		qc.State.CurrentHTLC = copyHTLC
	}

  if (msg.Incoming) {
    // They have to actually send you money
    if incomingDelta < 1 {
  			return fmt.Errorf("AssignHTLCDeltaSigHandler err: delta %d", incomingDelta)
  	}

    // perform minOutput check
  	theirNewOutputSize :=
  		qc.Value - (qc.State.MyAmt + int64(incomingDelta)) - qc.State.Fee

  	// check if this push is takes them below minimum output size
  	if theirNewOutputSize < minOutput {
  		qc.ClearToSend <- true
  		return fmt.Errorf(
  			"pushing %s reduces them too low; counterparty bal %s fee %s minOutput %s",
  			lnutil.SatoshiColor(int64(incomingDelta)),
  			lnutil.SatoshiColor(qc.Value-qc.State.MyAmt),
  			lnutil.SatoshiColor(qc.State.Fee),
  			lnutil.SatoshiColor(minOutput))
  	}
  } else {
    // You have to actually put money in the HTLC
    if incomingDelta > -1 {
  			return fmt.Errorf("AssignHTLCDeltaSigHandler err: delta %d", incomingDelta)
  	}

    // perform minOutput check
  	theirNewOutputSize :=
  		qc.Value - (qc.State.MyAmt + int64(incomingDelta)) - qc.State.Fee

  	// check if this push is takes them below minimum output size
  	if theirNewOutputSize < minOutput {
  		qc.ClearToSend <- true
  		return fmt.Errorf(
  			"pushing %s reduces them too low; counterparty bal %s fee %s minOutput %s",
  			lnutil.SatoshiColor(int64(incomingDelta)),
  			lnutil.SatoshiColor(qc.Value-qc.State.MyAmt),
  			lnutil.SatoshiColor(qc.State.Fee),
  			lnutil.SatoshiColor(minOutput))
  	}
  	// regardless of collision, drop amt
  	qc.State.MyAmt += int64(incomingDelta)
  }

  // update to the next state to verify
  qc.State.StateIdx++

	// verify sig for the next state. only save if this works
	err = qc.VerifySig(msg.Signature)
	if err != nil {
		return fmt.Errorf("DeltaSigHandler err %s", err.Error())
	}

	// (seems odd, but everything so far we still do in case of collision, so
	// only check here.  If it's a collision, set, save, send gapSigRev

	// save channel with new state, new sig, and positive delta set
	// and maybe collision; still haven't checked
	err = nd.SaveQchanState(qc)
	if err != nil {
		return fmt.Errorf("AssignHTLCDeltaSigHandler SaveQchanState err %s", err.Error())
	}

	if qc.State.Collision != 0 {
		err = nd.AssignHTLCSendGapSigRev(qc, msg.Incoming)
		if err != nil {
			return fmt.Errorf("AssignHTLCDeltaSigHandler SendGapSigRev err %s", err.Error())
		}
	} else { // saved to db, now proceed to create & sign their tx
		err = nd.AssignHTLCSendSigRev(qc, msg.Incoming)
		if err != nil {
			return fmt.Errorf("AssignHTLCDeltaSigHandler SendSigRev err %s", err.Error())
		}
	}
	return nil
}

func (nd *LitNode) AssignHTLCSendGapSigRev(q *Qchan, incoming bool) error {
	// state should already be set to the "gap" state; generate signature for n+1
	// the signature generation is similar to normal sigrev signing
	// in these "send_whatever" methods we don't modify and save to disk

	// state has been incremented in AssignHTLCDeltaSigHandler so n is the gap state
	// revoke n-1
	elk, err := q.ElkSnd.AtIndex(q.State.StateIdx - 1)
	if err != nil {
		return err
	}

	// send elkpoint for n+2
	n2ElkPoint, err := q.N2ElkPointForThem()
	if err != nil {
		return err
	}

	// go up to n+2 elkpoint for the signing
	q.State.ElkPoint = q.State.N2ElkPoint
	// state is already incremented from DeltaSigHandler, increment *again* for n+1
	// (note that we've moved n here.)
	q.State.StateIdx++
	// amt is delta (negative or zero depending on incoming or not) plus current amt (collision already added in)\
  q.State.MyAmt += int64(q.State.Delta)

	// sign state n+1
	sig, err := nd.SignState(q)
	if err != nil {
		return err
	}

	outMsg := lnutil.NewAssignHTLCGapSigRev(q.KeyGen.Step[3]&0x7fffffff, q.Op, !incoming, sig, *elk, n2ElkPoint)
	nd.OmniOut <- outMsg

	return nil
}

func (nd *LitNode) AssignHTLCSendSigRev(q *Qchan, incoming bool) error {

	// revoke n-1
	elk, err := q.ElkSnd.AtIndex(q.State.StateIdx - 1)
	if err != nil {
		return err
	}

	// state number and balance has already been updated if the incoming sig worked.
	// go to next elkpoint for signing
	// note that we have to keep the old elkpoint on disk for when the rev comes in
	q.State.ElkPoint = q.State.NextElkPoint
	// q.State.NextElkPoint = q.State.N2ElkPoint // not needed
	// n2elk invalid here

	sig, err := nd.SignState(q)
	if err != nil {
		return err
	}

	// send commitment elkrem point for next round of messages
	n2ElkPoint, err := q.N2ElkPointForThem()
	if err != nil {
		return err
	}

	outMsg := lnutil.NewAssignHTLCSigRev(q.KeyGen.Step[3]&0x7fffffff, q.Op, !incoming, sig, *elk, n2ElkPoint)
	nd.OmniOut <- outMsg

	return nil
}

func (nd *LitNode) AssignHTLCGapSigRevHandler(msg lnutil.AssignHTLCGapSigRevMsg, q *Qchan) error {

	// load qchan & state from DB
	err := nd.ReloadQchanState(q)
	if err != nil {
		return fmt.Errorf("GapSigRevHandler err %s", err.Error())
	}

	// check if we're supposed to get a AssignHTLCGapSigRev now. Collision should be set
  if (!msg.Incoming) {
    if q.State.Collision != 0 {
  		return fmt.Errorf(
  			"chan %d got AssignHTLCGapSigRev but collision = 0, delta = %d",
  			q.Idx(), q.State.Delta)
  	}
  }

	// stash for justice tx
	prevAmt := q.State.MyAmt - int64(q.State.Collision) // myAmt before collision

  // TODO.jesus?Check
  if (!msg.Incoming) {
    q.State.MyAmt += int64(q.State.Delta) // delta should be negative (if sending HTLC) or zero (if receiving HTLC)
  }
	q.State.Delta = q.State.Collision     // now delta is zero
	q.State.Collision = 0

	// verify elkrem and save it in ram
	err = q.AdvanceElkrem(&msg.Elk, msg.N2ElkPoint)
	if err != nil {
		return fmt.Errorf("AssignHTLCGapSigRevHandler err %s", err.Error())
		// ! non-recoverable error, need to close the channel here.
	}

	// go up to n+1 elkpoint for the sig verification
	stashElkPoint := q.State.ElkPoint
	q.State.ElkPoint = q.State.NextElkPoint

	// state is already incremented from AssignHTLCDeltaSigHandler, increment again for n+2
	// (note that we've moved n here.)
	q.State.StateIdx++

	// verify the sig
	err = q.VerifySig(msg.Signature)
	if err != nil {
		return fmt.Errorf("AssignHTLCGapSigRevHandler err %s", err.Error())
	}
	// go back to sequential elkpoints
	q.State.ElkPoint = stashElkPoint

	err = nd.SaveQchanState(q)
	if err != nil {
		return fmt.Errorf("AssignHTLCGapSigRevHandler err %s", err.Error())
	}
	err = nd.AssignHTLCSendREV(q, msg.Incoming)
	if err != nil {
		return fmt.Errorf("AssignHTLCGapSigRevHandler err %s", err.Error())
	}

	// for justice, have to create signature for n-2.  Remember the n-2 amount

	q.State.StateIdx -= 2
	q.State.MyAmt = prevAmt

	go func() {
		err = nd.BuildJusticeSig(q)
		if err != nil {
			fmt.Printf("AssignHTLCGapSigRevHandler BuildJusticeSig err %s", err.Error())
		}
	}()

	return nil
}

func (nd *LitNode) AssignHTLCSigRevHandler(msg lnutil.AssignHTLCSigRevMsg, qc *Qchan) error {

	// load qchan & state from DB
	err := nd.ReloadQchanState(qc)
	if err != nil {
		return fmt.Errorf("AssignHTLCSigRevHandler err %s", err.Error())
	}

  if (msg.Incoming) {
    if qc.State.Delta < 0 {
  		return fmt.Errorf("AssignHTLCSigRevHandler err: chan %d got AssignHTLCSigRev, expect AssignHTLCRev. delta %d",
  			qc.Idx(), qc.State.Delta)
  	}
  } else {
    if qc.State.Delta > 0 {
  		return fmt.Errorf("AssignHTLCSigRevHandler err: chan %d got AssignHTLCSigRev, expect AssignHTLCRev. delta %d",
  			qc.Idx(), qc.State.Delta)
  	}
  }

  // TODO.jesus?AddCheck
  // if (msg.Incoming) {
  //   if qc.State.Delta != 0 {
  // 		// re-send last rev; they probably didn't get it
  // 		return nd.AssignHTLCSendREV(qc)
  // 	}
  // } else {
  //   if qc.State.Delta == 0 {
  // 		// re-send last rev; they probably didn't get it
  // 		return nd.AssignHTLCSendREV(qc)
  // 	}
  // }

	if qc.State.Collision != 0 {
		return fmt.Errorf("chan %d got AssignHTLCSigRev, expect AssignHTLCGapSigRev delta %d col %d",
			qc.Idx(), qc.State.Delta, qc.State.Collision)
	}

	// stash previous amount here for watchtower sig creation
	prevAmt := qc.State.MyAmt

	qc.State.StateIdx++
  if (!msg.Incoming) {
    qc.State.MyAmt += int64(qc.State.Delta)
  }
	qc.State.Delta = 0

	// first verify sig.
	// (if elkrem ingest fails later, at least we close out with a bit more money)
	err = qc.VerifySig(msg.Signature)
	if err != nil {
		return fmt.Errorf("AssignHTLCSigRev err %s", err.Error())
	}

	// verify elkrem and save it in ram
	err = qc.AdvanceElkrem(&msg.Elk, msg.N2ElkPoint)
	if err != nil {
		return fmt.Errorf("AssignHTLCSigRev err %s", err.Error())
		// ! non-recoverable error, need to close the channel here.
	}
	// if the elkrem failed but sig didn't... we should update the DB to reflect
	// that and try to close with the incremented amount, why not.
	// TODO Implement that later though.

	// all verified; Save finished state to DB, puller is pretty much done.
	err = nd.SaveQchanState(qc)
	if err != nil {
		return fmt.Errorf("AssignHTLCSigRev err %s", err.Error())
	}

	fmt.Printf("AssignHTLCSigRev OK, state %d, will send AssignHTLCRev\n", qc.State.StateIdx)
	err = nd.AssignHTLCSendREV(qc, msg.Incoming)
	if err != nil {
		return fmt.Errorf("AssignHTLCSigRev err %s", err.Error())
	}

	// now that we've saved & sent everything, before ending the function, we
	// go BACK to create a txid/sig pair for watchtower.  This feels like a kindof
	// weird way to do it.  Maybe there's a better way.

	qc.State.StateIdx--
	qc.State.MyAmt = prevAmt

	go func() {
		err = nd.BuildJusticeSig(qc)
		if err != nil {
			fmt.Printf("AssignHTLCSigRevHandler BuildJusticeSig err %s", err.Error())
		}
	}()

	// done updating channel, no new messages expected.  Set clear to send
	qc.ClearToSend <- true

	return nil
}

func (nd *LitNode) AssignHTLCSendREV(q *Qchan, incoming bool) error {
	// revoke previous already built state
	elk, err := q.ElkSnd.AtIndex(q.State.StateIdx - 1)
	if err != nil {
		return err
	}
	// send commitment elkrem point for next round of messages
	n2ElkPoint, err := q.N2ElkPointForThem()
	if err != nil {
		return err
	}

	outMsg := lnutil.NewAssignHTLCRevMsg(q.Peer(), q.Op, !incoming, *elk, n2ElkPoint)
	nd.OmniOut <- outMsg

	return err
}

func (nd *LitNode) AssignHTLCRevHandler(msg lnutil.AssignHTLCRevMsg, qc *Qchan) error {

	// load qchan & state from DB
	err := nd.ReloadQchanState(qc)
	if err != nil {
		return fmt.Errorf("AssignHTLCRevHandler err %s", err.Error())
	}

  // TODO.jesus?AddCheck for else case
  // if (msg.Incoming) {
  //   // check if there's nothing for them to revoke
  //   if qc.State.Delta == 0 {
  // 		return fmt.Errorf("got AssignHTLCRev, expected AssignHTLCDeltaSig, ignoring.")
  // 	}
  //
  //   // maybe this is an unexpected rev, asking us for a rev repeat
  //   if qc.State.Delta < 0 {
  // 		fmt.Printf("got AssignHTLCRev, expected AssignHTLCSigRev.  Re-sending last AssignHTLCRev.\n")
  // 		return nd.AssignHTLCSendREV(qc)
  // 	}
  // }

	// verify elkrem
	err = qc.AdvanceElkrem(&msg.Elk, msg.N2ElkPoint)
	if err != nil {
		fmt.Printf(" ! non-recoverable error, need to close the channel here.\n")
		return fmt.Errorf("AssignHTLCRevHandler err %s", err.Error())
	}
	prevAmt := qc.State.MyAmt - int64(qc.State.Delta)
	qc.State.Delta = 0

	// save to DB (new elkrem & point, delta zeroed)
	err = nd.SaveQchanState(qc)
	if err != nil {
		return fmt.Errorf("AssignHTLCRevHandler err %s", err.Error())
	}

	// after saving cleared updated state, go back to previous state and build
	// the justice signature
	qc.State.StateIdx--      // back one state
	qc.State.MyAmt = prevAmt // use stashed previous state amount
	go func() {
		err = nd.BuildJusticeSig(qc)
		if err != nil {
			fmt.Printf("AssignHTLCRevHandler BuildJusticeSig err %s", err.Error())
		}
	}()

	// got rev, assert clear to send
	qc.ClearToSend <- true

	fmt.Printf("AssignHTLCRev OK, state %d all clear.\n", qc.State.StateIdx)
	return nil
}