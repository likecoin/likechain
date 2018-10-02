package deposit

import (
	"fmt"
	"testing"

	"github.com/likecoin/likechain/abci/context"
	"github.com/likecoin/likechain/abci/response"
	"github.com/likecoin/likechain/abci/types"

	. "github.com/smartystreets/goconvey/convey"
)

func wrapDepositTransaction(tx *types.DepositTransaction) *types.Transaction {
	return &types.Transaction{
		Tx: &types.Transaction_DepositTx{
			DepositTx: tx,
		},
	}
}

func TestCheckAndDeliverDeposit(t *testing.T) {
	appCtx := context.NewMock()
	state := appCtx.GetMutableState()

	SkipConvey("Given a Deposit Transaction", t, func() {

		Convey("If it is a valid transaction", func() {
			appCtx.Reset()
			rawTx := wrapDepositTransaction(&types.DepositTransaction{
				// TODO
			})

			Convey("CheckTx should return Code 0", func() {
				res := checkDeposit(state, rawTx)

				So(res.Code, ShouldEqual, 0)
			})

			Convey("DeliverTx should return Code 0", func() {
				res := deliverDeposit(state, rawTx, []byte{})

				So(res.Code, ShouldEqual, 0)

				Convey("Should be able to query the transaction info afterwards", func() {
					_ = res.Data // TODO: ID
					// TODO: query
				})
			})
		})

		Convey("If it is an invalid address format", func() {
			appCtx.Reset()

			rawTx := wrapDepositTransaction(&types.DepositTransaction{
				// TODO
			})

			code := response.DepositCheckTxInvalidFormat.Code
			Convey(fmt.Sprintf("CheckTx should return Code %d", code), func() {
				res := checkDeposit(state, rawTx)

				So(res.Code, ShouldEqual, code)
			})

			code = response.DepositDeliverTxInvalidFormat.Code
			Convey(fmt.Sprintf("DeliverTx should return Code %d", code), func() {
				res := deliverDeposit(state, rawTx, []byte{})

				So(res.Code, ShouldEqual, code)
			})
		})

		Convey("If it is a replayed transaction", func() {
			appCtx.Reset()

			rawTx := wrapDepositTransaction(&types.DepositTransaction{
				// TODO
			})

			code := response.DepositCheckTxDuplicated.Code
			Convey(fmt.Sprintf("CheckTx should return Code %d", code), func() {
				res := checkDeposit(state, rawTx)

				So(res.Code, ShouldEqual, code)
			})

			code = response.DepositDeliverTxDuplicated.Code
			Convey(fmt.Sprintf("DeliverTx should return Code %d", code), func() {
				res := deliverDeposit(state, rawTx, []byte{})

				So(res.Code, ShouldEqual, code)
			})
		})
	})
}

func TestValidateDepositTransactionFormat(t *testing.T) {
	SkipConvey("Given a Deposit transaction in valid format", t, func() {
		tx := &types.DepositTransaction{} // TODO

		Convey("The transaction should pass the validation", func() {
			So(validateDepositTransactionFormat(tx), ShouldBeTrue)
		})
	})

	Convey("Given a Deposit transaction in invalid format", t, func() {
		tx := &types.DepositTransaction{} // TODO

		Convey("The transaction should not pass the validation", func() {
			So(validateDepositTransactionFormat(tx), ShouldBeFalse)
		})
	})
}

func TestDeposit(t *testing.T) {
	appCtx := context.NewMock()
	state := appCtx.GetMutableState()

	SkipConvey("Given a valid Deposit transaction", t, func() {
		tx := &types.DepositTransaction{} // TODO

		Convey("The transaction should be pass", func() {
			deposit(state, tx)
			// TODO: checking
		})

		Convey("But the same Deposit transaction cannot be replayed", func() {
			deposit(state, tx)
			// TODO: checking
		})
	})

	SkipConvey("Given an invalid Deposit transaction", t, func() {
		appCtx.Reset()
		tx := &types.DepositTransaction{} // TODO

		Convey("The transaction should not be pass if receiver not exist", func() {
			deposit(state, tx)
			// TODO: checking
		})
	})
}
