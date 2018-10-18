package transfer

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/likecoin/likechain/abci/account"
	"github.com/likecoin/likechain/abci/context"
	"github.com/likecoin/likechain/abci/handlers/table"
	logger "github.com/likecoin/likechain/abci/log"
	"github.com/likecoin/likechain/abci/response"
	"github.com/likecoin/likechain/abci/types"
	"github.com/likecoin/likechain/abci/utils"
	"github.com/sirupsen/logrus"
)

const maxRemarkSize = 4096

var log = logger.L

func logTx(tx *types.TransferTransaction) *logrus.Entry {
	return log.WithField("tx", tx)
}

func checkTransfer(state context.IImmutableState, rawTx *types.Transaction) response.R {
	tx := rawTx.GetTransferTx()
	if tx == nil {
		log.Panic("Expect TransferTx but got nil")
	}

	if !validateTransferTransactionFormat(state, tx) {
		logTx(tx).Info(response.TransferCheckTxInvalidFormat.Info)
		return response.TransferCheckTxInvalidFormat
	}

	senderID := account.IdentifierToLikeChainID(state, tx.From)
	if senderID == nil {
		logTx(tx).Info(response.TransferCheckTxSenderNotRegistered.Info)
		return response.TransferCheckTxSenderNotRegistered
	}

	if !validateTransferSignature(state, tx) {
		logTx(tx).Info(response.TransferCheckTxInvalidSignature.Info)
		return response.TransferCheckTxInvalidSignature
	}

	nextNonce := account.FetchNextNonce(state, senderID)
	if tx.Nonce > nextNonce {
		logTx(tx).Info(response.TransferCheckTxInvalidNonce.Info)
		return response.TransferCheckTxInvalidNonce
	} else if tx.Nonce < nextNonce {
		logTx(tx).Info(response.TransferCheckTxDuplicated.Info)
		return response.TransferCheckTxDuplicated
	}

	senderBalance := account.FetchBalance(state, senderID.ToIdentifier())
	total := tx.Fee.ToBigInt()
	for _, target := range tx.ToList {
		if target.To.GetLikeChainID() != nil {
			targetID := account.IdentifierToLikeChainID(state, target.To)
			if targetID == nil {
				logTx(tx).
					WithField("to", target.To.ToString()).
					Info(response.TransferCheckTxInvalidReceiver.Info)
				return response.TransferCheckTxInvalidReceiver
			}
		}
		amount := target.Value.ToBigInt()
		total.Add(total, amount)
		if senderBalance.Cmp(total) < 0 {
			logTx(tx).
				WithField("total", total.String()).
				WithField("balance", senderBalance.String()).
				Info(response.TransferCheckTxNotEnoughBalance.Info)
			return response.TransferCheckTxNotEnoughBalance
		}
	}

	return response.Success
}

func deliverTransfer(
	state context.IMutableState,
	rawTx *types.Transaction,
	txHash []byte,
) response.R {
	r := deliver(state, rawTx, txHash)

	var status types.TxStatus
	if r.Code != 0 {
		status = types.TxStatusFail
	} else {
		status = types.TxStatusSuccess
	}

	prevStatus := GetStatus(state, txHash)
	if prevStatus == types.TxStatusNotSet {
		SetStatus(state, txHash, status)
	}

	return r
}

func deliver(
	state context.IMutableState,
	rawTx *types.Transaction,
	txHash []byte,
) response.R {
	tx := rawTx.GetTransferTx()
	if tx == nil {
		log.Panic("Expect TransferTx but got nil")
	}

	if !validateTransferTransactionFormat(state, tx) {
		logTx(tx).Info(response.TransferDeliverTxInvalidFormat.Info)
		return response.TransferDeliverTxInvalidFormat
	}

	senderID := account.IdentifierToLikeChainID(state, tx.From)
	if senderID == nil {
		logTx(tx).Info(response.TransferDeliverTxSenderNotRegistered.Info)
		return response.TransferDeliverTxSenderNotRegistered
	}

	if !validateTransferSignature(state, tx) {
		logTx(tx).Info(response.TransferDeliverTxInvalidSignature.Info)
		return response.TransferDeliverTxInvalidSignature
	}

	nextNonce := account.FetchNextNonce(state, senderID)
	if tx.Nonce > nextNonce {
		logTx(tx).Info(response.TransferDeliverTxInvalidNonce.Info)
		return response.TransferDeliverTxInvalidNonce
	} else if tx.Nonce < nextNonce {
		logTx(tx).Info(response.TransferDeliverTxDuplicated.Info)
		return response.TransferDeliverTxDuplicated
	}

	account.IncrementNextNonce(state, senderID)

	senderIden := senderID.ToIdentifier()
	senderBalance := account.FetchBalance(state, senderIden)
	total := tx.Fee.ToBigInt()
	transfers := make(map[*types.Identifier]*big.Int, len(tx.ToList))
	for _, target := range tx.ToList {
		if target.To.GetLikeChainID() != nil {
			targetID := account.IdentifierToLikeChainID(state, target.To)
			if targetID == nil {
				logTx(tx).
					WithField("to", target.To.ToString()).
					Info(response.TransferDeliverTxInvalidReceiver.Info)
				return response.TransferDeliverTxInvalidReceiver
			}
		}
		amount := target.Value.ToBigInt()
		total.Add(total, amount)
		if senderBalance.Cmp(total) < 0 {
			logTx(tx).
				WithField("total", total.String()).
				WithField("balance", senderBalance.String()).
				Info(response.TransferDeliverTxNotEnoughBalance.Info)
			return response.TransferDeliverTxNotEnoughBalance
		}
		targetIden := target.To
		targetID := account.IdentifierToLikeChainID(state, target.To)
		if targetID != nil {
			targetIden = targetID.ToIdentifier()
		}

		transfers[targetIden] = amount
	}

	for to, amount := range transfers {
		account.AddBalance(state, to, amount)
	}
	account.MinusBalance(state, senderIden, total)

	return response.Success
}

func validateTransferSignature(state context.IImmutableState, tx *types.TransferTransaction) bool {
	hashedMsg := tx.GenerateSigningMessageHash()
	sigAddr, err := utils.RecoverSignature(hashedMsg, tx.Sig)
	if err != nil {
		log.WithError(err).Info("Unable to recover signature when validating signature")
		return false
	}

	senderAddr := tx.From.GetAddr()
	if senderAddr != nil {
		if senderAddr.ToEthereum() == sigAddr {
			return true
		}
		log.WithFields(logrus.Fields{
			"tx_addr":  senderAddr.ToHex(),
			"sig_addr": sigAddr.Hex(),
		}).Info("Recovered address is not match")
	} else {
		id := tx.From.GetLikeChainID()
		if id != nil {
			if account.IsLikeChainIDHasAddress(state, id, sigAddr) {
				return true
			}
			log.WithFields(logrus.Fields{
				"likechain_id": id.ToString(),
				"sig_addr":     sigAddr.Hex(),
			}).Info("Recovered address is not bind to the LikeChain ID of the sender")
		}
	}

	return false
}

func validateTransferTransactionFormat(state context.IImmutableState, tx *types.TransferTransaction) bool {
	if !tx.From.IsValidFormat() {
		log.Debug("Invalid sender format in transfer transaction")
		return false
	}

	if len(tx.ToList) > 0 {
		for _, target := range tx.ToList {
			if !target.IsValidFormat() {
				log.Debug("Invalid receiver format in transfer transaction")
				return false
			}
			if len(target.Remark) > maxRemarkSize {
				log.WithField("size", len(target.Remark)).
					Debug(fmt.Sprintf("Size of the remark exceeds %dB", maxRemarkSize))
				return false
			}
		}
	} else {
		log.Debug("No receiver in transfer transaction")
		return false
	}

	if !tx.Sig.IsValidFormat() {
		log.Debug("Invalid signature format in transfer transaction")
		return false
	}

	return true
}

func getStatusKey(txHash []byte) []byte {
	return utils.DbTxHashKey(txHash, "status")
}

// GetStatus returns transaction status by txHash
func GetStatus(state context.IImmutableState, txHash []byte) types.TxStatus {
	_, statusBytes := state.ImmutableStateTree().Get(getStatusKey(txHash))
	return types.BytesToTxStatus(statusBytes)
}

// SetStatus set the transaction status of the given txHash
func SetStatus(
	state context.IMutableState,
	txHash []byte,
	status types.TxStatus,
) {
	state.MutableStateTree().Set(getStatusKey(txHash), status.Bytes())
}

func init() {
	log.Info("Init transfer handlers")
	_type := reflect.TypeOf((*types.Transaction_TransferTx)(nil))
	table.RegisterCheckTxHandler(_type, checkTransfer)
	table.RegisterDeliverTxHandler(_type, deliverTransfer)
}
