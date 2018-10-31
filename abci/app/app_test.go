package app

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/likecoin/likechain/abci/account"
	appConf "github.com/likecoin/likechain/abci/config"
	"github.com/likecoin/likechain/abci/context"
	"github.com/likecoin/likechain/abci/query"
	"github.com/likecoin/likechain/abci/response"
	"github.com/likecoin/likechain/abci/txs"
	"github.com/likecoin/likechain/abci/types"

	"github.com/tendermint/iavl"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/tmhash"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGeneral(t *testing.T) {
	Convey("At initial state", t, func() {
		mockCtx := context.NewMock()
		app := &LikeChainApplication{
			ctx: mockCtx.ApplicationContext,
		}
		app.InitChain(abci.RequestInitChain{})
		Convey("Given an invalid Transaction", func() {
			rawTx := make([]byte, 20)
			Convey("CheckTx should return code 1", func() {
				r := app.CheckTx(rawTx)
				So(r.Code, ShouldEqual, 1)
				Convey("DeliverTx should return code 1", func() {
					r := app.DeliverTx(rawTx)
					So(r.Code, ShouldEqual, 1)
				})
			})
		})
	})
}

func TestRegistration(t *testing.T) {
	Convey("At initial state", t, func() {
		mockCtx := context.NewMock()
		app := &LikeChainApplication{
			ctx: mockCtx.ApplicationContext,
		}
		app.InitChain(abci.RequestInitChain{})
		Convey("Given a valid RegisterTransaction", func() {
			rawTx := txs.RawRegisterTx("0x539c17e9e5fd1c8e3b7506f4a7d9ba0a0677eae9", "65e6d31224fbcec8e41251d7b014e569d4a94c866227637c6b1fcf75a4505f241b2009557e79d5879a8bfbbb5dec86205c3481ed3042ad87f0643778022f54141b")
			Convey("The registration should succeed", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				Convey("Duplicated registration in the same block should fail during deliverTx", func() {
					checkTxResDup := app.CheckTx(rawTx)
					So(checkTxResDup.Code, ShouldEqual, response.RegisterDuplicated.ToResponseCheckTx().Code)
					deliverTxResDup := app.DeliverTx(rawTx)
					So(deliverTxResDup.Code, ShouldEqual, response.RegisterDuplicated.ToResponseDeliverTx().Code)
				})
				app.EndBlock(abci.RequestEndBlock{
					Height: 1,
				})
				app.Commit()
				likeChainID := deliverTxRes.Data
				Convey("Then query tx_state should return success", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "success")
				})
				Convey("Query account_info using address should return the corresponding info", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte("0x539c17e9e5fd1c8e3b7506f4a7d9ba0a0677eae9"),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainID)
					So(accountInfo.Balance.Cmp(big.NewInt(0)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("Query account_info using returned LikeChain ID should return the corresponding info", func() {
					likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainID)
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainID)
					So(accountInfo.Balance.Cmp(big.NewInt(0)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("Query address_info should return the corresponding info", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "address_info",
						Data: []byte("0x539c17e9e5fd1c8e3b7506f4a7d9ba0a0677eae9"),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainID)
					So(accountInfo.Balance.Cmp(big.NewInt(0)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("But repeated registration should fail", func() {
					checkTxRes = app.CheckTx(rawTx)
					So(checkTxRes.Code, ShouldEqual, response.RegisterDuplicated.ToResponseCheckTx().Code)
					deliverTxRes = app.DeliverTx(rawTx)
					So(deliverTxRes.Code, ShouldEqual, response.RegisterDuplicated.ToResponseDeliverTx().Code)
					Convey("Query tx_state should still return success", func() {
						txHash := tmhash.Sum(rawTx)
						queryRes := app.Query(abci.RequestQuery{
							Path: "tx_state",
							Data: txHash,
						})
						So(queryRes.Code, ShouldEqual, response.Success.Code)
						txStateRes := query.GetTxStateRes(queryRes.Value)
						So(txStateRes, ShouldNotBeNil)
						So(txStateRes.Status, ShouldEqual, "success")
					})
				})
			})
		})

		Convey("Given a RegisterTransaction with other's signature", func() {
			rawTx := txs.RawRegisterTx("0x539c17e9e5fd1c8e3b7506f4a7d9ba0a0677eae9", "b287bb3c420155326e0a7fe3a66fed6c397a4bdb5ddcd54960daa0f06c1fbf06300e862dbd3ae3daeae645630e66962b81cf6aa9ffb258aafde496e0310ab8551c")
			Convey("The registration should fail", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.RegisterInvalidSignature.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.RegisterInvalidSignature.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 1,
				})
				app.Commit()
				Convey("Then query tx_state should return fail", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "fail")
				})
			})
		})

		Convey("Given a RegisterTransaction with invalid signature", func() {
			rawTx := txs.RawRegisterTx("0x539c17e9e5fd1c8e3b7506f4a7d9ba0a0677eae9", "65e6d31224fbcec8e41251d7b014e569d4a94c866227637c6b1fcf75a4505f241b2009557e79d5879a8bfbbb5dec86205c3481ed3042ad87f0643778022f541400")
			Convey("The registration should fail", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.RegisterInvalidSignature.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.RegisterInvalidSignature.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 1,
				})
				app.Commit()
				Convey("Then query tx_state should return fail", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "fail")
				})
			})
		})
	})
}

func TestTransfer(t *testing.T) {
	Convey("Given accounts A, B, C with 100, 200, 300 balance each", t, func() {
		mockCtx := context.NewMock()
		app := &LikeChainApplication{
			ctx: mockCtx.ApplicationContext,
		}
		app.InitChain(abci.RequestInitChain{})

		likeChainIDs := [][]byte{}
		regInfos := []struct {
			Addr string
			Sig  string
		}{
			{"0x539c17e9e5fd1c8e3b7506f4a7d9ba0a0677eae9", "65e6d31224fbcec8e41251d7b014e569d4a94c866227637c6b1fcf75a4505f241b2009557e79d5879a8bfbbb5dec86205c3481ed3042ad87f0643778022f54141b"},
			{"0x833a907efe57af3040039c90f4a59946a0bb3d47", "fb141eca7550c8f6d1f37b20536b06327ba29537a6178ea39e9d7747abdc8c2c4daa4ab23accf2157a2eb5ec1eb54ee68159c5b39f7f4ac17087fd71afd374121b"},
			{"0xaa2f5b6ae13ba7a3d466ffce8cd390519337aade", "e906aaf924d636c9b03160d358ec9a20b2b79770e807e84f4cf7f274149ff0b1185b8508adf8cbbc0436b3215cb6e77fea84e97340cbdacd2bcc0bac4a374b441b"},
		}

		for n, regInfo := range regInfos {
			rawTx := txs.RawRegisterTx(regInfo.Addr, regInfo.Sig)
			deliverTxRes := app.DeliverTx(rawTx)
			So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
			likeChainID := deliverTxRes.Data
			likeChainIDs = append(likeChainIDs, likeChainID)
			account.SaveBalance(mockCtx.GetMutableState(), types.ID(likeChainID), big.NewInt(int64(n+1)*100))
		}
		app.EndBlock(abci.RequestEndBlock{
			Height: 1,
		})
		app.Commit()

		for i, likeChainIDBase64 := range []string{"bDH8FUIuutKKr5CJwwZwL2dUC1M=", "hZ8Rt1VppOsElsUTj9QsxSrujPU=", "1MaeSeg6YEf0bkKy0FOh8MbnDqQ="} {
			So(likeChainIDs[i], ShouldResemble, types.IDStr(likeChainIDBase64)[:])
		}

		for n, likeChainID := range likeChainIDs {
			likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainID)
			queryRes := app.Query(abci.RequestQuery{
				Path: "account_info",
				Data: []byte(likeChainIDBase64),
			})
			So(queryRes.Code, ShouldEqual, response.Success.Code)
			accountInfo := query.GetAccountInfoRes(queryRes.Value)
			So(accountInfo, ShouldNotBeNil)
			So(accountInfo.Balance.Cmp(big.NewInt(int64(n+1)*100)), ShouldBeZeroValue)
			So(accountInfo.NextNonce, ShouldEqual, 1)
		}

		Convey("Given a TransferTransaction from A to B value 1", func() {
			outputs := []txs.TransferOutput{
				{
					To:    types.ID(likeChainIDs[1]),
					Value: types.NewBigInt(1),
				},
			}
			rawTx := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 1, "ab1a659bd26576e8afeeba1ff3885da74c3c1088202770b029f4f2c555bd874811063768b93113fb66e4545b08e9030e94f83fb5c8484422107e8434f77c3c851c")
			Convey("The transfer should succeed", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query account_info by LikeChain ID should return the correct balances and nextNonce", func() {
					likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(99)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)

					likeChainIDBase64 = base64.StdEncoding.EncodeToString(likeChainIDs[1])
					queryRes = app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[1])
					So(accountInfo.Balance.Cmp(big.NewInt(201)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("Then query account_info by Ethereum address should return the correct balances and nextNonce", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(regInfos[0].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(99)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)

					queryRes = app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(regInfos[1].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[1])
					So(accountInfo.Balance.Cmp(big.NewInt(201)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("Then query address_info should return the correct balances and nextNonce", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "address_info",
						Data: []byte(regInfos[0].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(99)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)

					queryRes = app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(regInfos[1].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[1])
					So(accountInfo.Balance.Cmp(big.NewInt(201)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("Then query tx_state should return success", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "success")
				})
				Convey("But repeated transfer with the same transaction should fail", func() {
					checkTxRes := app.CheckTx(rawTx)
					So(checkTxRes.Code, ShouldEqual, response.TransferDuplicated.ToResponseCheckTx().Code)
					deliverTxRes := app.DeliverTx(rawTx)
					So(deliverTxRes.Code, ShouldEqual, response.TransferDuplicated.ToResponseDeliverTx().Code)
					Convey("Then query tx_state should still return success", func() {
						txHash := tmhash.Sum(rawTx)
						queryRes := app.Query(abci.RequestQuery{
							Path: "tx_state",
							Data: txHash,
						})
						So(queryRes.Code, ShouldEqual, response.Success.Code)
						txStateRes := query.GetTxStateRes(queryRes.Value)
						So(txStateRes, ShouldNotBeNil)
						So(txStateRes.Status, ShouldEqual, "success")
					})
					Convey("Then query account_info by LikeChain ID should return the correct balances and nextNonce", func() {
						likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
						queryRes := app.Query(abci.RequestQuery{
							Path: "account_info",
							Data: []byte(likeChainIDBase64),
						})
						So(queryRes.Code, ShouldEqual, response.Success.Code)
						accountInfo := query.GetAccountInfoRes(queryRes.Value)
						So(accountInfo, ShouldNotBeNil)
						So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
						So(accountInfo.Balance.Cmp(big.NewInt(99)), ShouldBeZeroValue)
						So(accountInfo.NextNonce, ShouldEqual, 2)

						likeChainIDBase64 = base64.StdEncoding.EncodeToString(likeChainIDs[1])
						queryRes = app.Query(abci.RequestQuery{
							Path: "account_info",
							Data: []byte(likeChainIDBase64),
						})
						So(queryRes.Code, ShouldEqual, response.Success.Code)
						accountInfo = query.GetAccountInfoRes(queryRes.Value)
						So(accountInfo, ShouldNotBeNil)
						So(accountInfo.ID, ShouldResemble, likeChainIDs[1])
						So(accountInfo.Balance.Cmp(big.NewInt(201)), ShouldBeZeroValue)
						So(accountInfo.NextNonce, ShouldEqual, 1)
					})
				})
			})
		})

		Convey("Given a TransferTransaction from A's Ethereum address to B's Ethereum address with value 1", func() {
			outputs := []txs.TransferOutput{
				{
					To:    types.Addr(regInfos[1].Addr),
					Value: types.NewBigInt(1),
				},
			}
			rawTx := txs.RawTransferTx(types.Addr(regInfos[0].Addr), outputs, types.NewBigInt(0), 1, "77656a44610efb227eaf1a1cffa05ebb43cb323d755825771cda47b3823ac19029727f334bc2ccdb1983c917944c5781f3a6a3b1ddd2cff5ee2d20f98e68de291c")
			Convey("The transfer should succeed", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query account_info by LikeChain ID address should return the correct balance", func() {
					likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(99)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)

					likeChainIDBase64 = base64.StdEncoding.EncodeToString(likeChainIDs[1])
					queryRes = app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[1])
					So(accountInfo.Balance.Cmp(big.NewInt(201)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("Then query account_info by Ethereum address should return the correct balance", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(regInfos[0].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(99)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)

					queryRes = app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(regInfos[1].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[1])
					So(accountInfo.Balance.Cmp(big.NewInt(201)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("Then query address_info by Ethereum address should return the correct balance", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "address_info",
						Data: []byte(regInfos[0].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(99)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)

					queryRes = app.Query(abci.RequestQuery{
						Path: "address_info",
						Data: []byte(regInfos[1].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[1])
					So(accountInfo.Balance.Cmp(big.NewInt(201)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("Then query tx_state should return success", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "success")
				})
			})
		})

		Convey("Given a TransferTransaction from A's LikeChain ID to B's Ethereum address (with value 1) and C's LikeChain ID (with value 2)", func() {
			outputs := []txs.TransferOutput{
				{
					To:    types.Addr(regInfos[1].Addr),
					Value: types.NewBigInt(1),
				},
				{
					To:    types.ID(likeChainIDs[2]),
					Value: types.NewBigInt(2),
				},
			}
			rawTx := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 1, "ba7304e7c4622486a003195c6db35e80fda1a4ef6ae606c1cddeebde903002ae0daa560610c6720169e222d39823c3f1ffbd92f72000ef07f5e9e43c796ffc9a1b")
			Convey("The transfer should succeed", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query account_info by LikeChain ID address should return the correct balance", func() {
					likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(97)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)

					likeChainIDBase64 = base64.StdEncoding.EncodeToString(likeChainIDs[1])
					queryRes = app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[1])
					So(accountInfo.Balance.Cmp(big.NewInt(201)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)

					likeChainIDBase64 = base64.StdEncoding.EncodeToString(likeChainIDs[2])
					queryRes = app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[2])
					So(accountInfo.Balance.Cmp(big.NewInt(302)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("Then query account_info by Ethereum address should return the correct balance", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(regInfos[0].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(97)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)

					queryRes = app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(regInfos[1].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[1])
					So(accountInfo.Balance.Cmp(big.NewInt(201)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)

					queryRes = app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(regInfos[2].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[2])
					So(accountInfo.Balance.Cmp(big.NewInt(302)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("Then query address_info should return the correct balance", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "address_info",
						Data: []byte(regInfos[0].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(97)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)

					queryRes = app.Query(abci.RequestQuery{
						Path: "address_info",
						Data: []byte(regInfos[1].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[1])
					So(accountInfo.Balance.Cmp(big.NewInt(201)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)

					queryRes = app.Query(abci.RequestQuery{
						Path: "address_info",
						Data: []byte(regInfos[2].Addr),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[2])
					So(accountInfo.Balance.Cmp(big.NewInt(302)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("Then query tx_state should return success", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "success")
				})
			})
		})

		Convey("Given a TransferTransaction from unregistered Ethereum address", func() {
			outputs := []txs.TransferOutput{
				{
					To:    types.Addr(regInfos[1].Addr),
					Value: types.NewBigInt(1),
				},
			}
			rawTx := txs.RawTransferTx(types.Addr("0xf6c45b1c4b73ccaeb1d9a37024a6b9fa711d7e7e"), outputs, types.NewBigInt(0), 1, "e0a0f6ece93840cad98c84f854eeaf8defaf327d18504ed04ad2a9be951402974762a229ecfc01850462d1f887a04c5db02bae90b4bb53a46394586ea15ef8f51b")
			Convey("The transfer should fail with SenderNotRegistered", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.TransferSenderNotRegistered.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.TransferSenderNotRegistered.ToResponseDeliverTx().Code)
				Convey("Then query tx_state should return fail", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "fail")
				})
			})
		})

		Convey("Given a TransferTransaction to unregistered LikeChain ID receiver(s)", func() {
			outputs := []txs.TransferOutput{
				{
					To:    types.Addr(regInfos[1].Addr),
					Value: types.NewBigInt(1),
				},
				{
					To:    types.IDStr("j/FYH9yZaCgTbAuhvdvk+op9Vas="),
					Value: types.NewBigInt(1),
				},
			}
			rawTx := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 1, "98b90aea28c30af6cbbbdecf4dcd37030f5d74fde50356dfdc106bd815a65cac60cf32e98a56961facecf4ff0353b7f264c30429886d32322568929300f01c781c")
			Convey("The transfer should fail with InvalidReceiver", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.TransferInvalidReceiver.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.TransferInvalidReceiver.ToResponseDeliverTx().Code)
				Convey("Then query tx_state should return fail", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "fail")
				})
				Convey("Query account_info should return unchanged balance and increased nonce", func() {
					likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(100)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)
				})
			})
		})

		Convey("Given a TransferTransaction to unregistered Ethereum address", func() {
			outputs := []txs.TransferOutput{
				{
					To:    types.Addr("0xf6c45b1c4b73ccaeb1d9a37024a6b9fa711d7e7e"),
					Value: types.NewBigInt(1),
				},
			}
			rawTx := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 1, "c0471fd0c5892dd7eb84548ec6e17df171b53423d1117459a10304309c287f7c6a319ddc2097b08768c263b0acb6bfd145ebfa3ecf0ef674ea7520ac27605e731c")
			Convey("The transfer should succeed", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query account_info for sender's account should return the correct balance", func() {
					likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(99)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)
				})
				Convey("Query account_info for receiver's address should return nothing before registration", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte("0xf6c45b1c4b73ccaeb1d9a37024a6b9fa711d7e7e"),
					})
					So(queryRes.Code, ShouldEqual, response.QueryInvalidIdentifier.Code)
				})
				Convey("Query address_info for receiver's address should return the correct balance", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "address_info",
						Data: []byte("0xf6c45b1c4b73ccaeb1d9a37024a6b9fa711d7e7e"),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldBeEmpty)
					So(accountInfo.Balance.Cmp(big.NewInt(1)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 0)
				})
				Convey("Then query tx_state should return success", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "success")
				})
				Convey("Registration for the receiver's address should succeed", func() {
					rawTx := txs.RawRegisterTx("0xf6c45b1c4b73ccaeb1d9a37024a6b9fa711d7e7e", "5221a47f0c1042f67951e28c513634190a7c4d77703a642d495ac5ef6397c4ec4d6ab2f7d1cda7c05f8e61d781aa2a4fa6e98c4382f741c4a7ab8e4de1d3fee31c")
					checkTxRes := app.CheckTx(rawTx)
					So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
					deliverTxRes := app.DeliverTx(rawTx)
					So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
					app.EndBlock(abci.RequestEndBlock{
						Height: 3,
					})
					app.Commit()
					likeChainID := deliverTxRes.Data
					Convey("The receiving Ethereum address should have balance after registration", func() {
						likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainID)
						queryRes := app.Query(abci.RequestQuery{
							Path: "account_info",
							Data: []byte(likeChainIDBase64),
						})
						So(queryRes.Code, ShouldEqual, response.Success.Code)
						accountInfo := query.GetAccountInfoRes(queryRes.Value)
						So(accountInfo, ShouldNotBeNil)
						So(accountInfo.Balance.Cmp(big.NewInt(1)), ShouldBeZeroValue)
						So(accountInfo.NextNonce, ShouldEqual, 1)

						queryRes = app.Query(abci.RequestQuery{
							Path: "account_info",
							Data: []byte("0xf6c45b1c4b73ccaeb1d9a37024a6b9fa711d7e7e"),
						})
						So(queryRes.Code, ShouldEqual, response.Success.Code)
						accountInfo = query.GetAccountInfoRes(queryRes.Value)
						So(accountInfo, ShouldNotBeNil)
						So(accountInfo.Balance.Cmp(big.NewInt(1)), ShouldBeZeroValue)
						So(accountInfo.NextNonce, ShouldEqual, 1)
						Convey("Query address_info for receiver's address should return the registered LikeCoin ID", func() {
							queryRes := app.Query(abci.RequestQuery{
								Path: "address_info",
								Data: []byte("0xf6c45b1c4b73ccaeb1d9a37024a6b9fa711d7e7e"),
							})
							So(queryRes.Code, ShouldEqual, response.Success.Code)
							accountInfo := query.GetAccountInfoRes(queryRes.Value)
							So(accountInfo, ShouldNotBeNil)
							So(accountInfo.Balance.Cmp(big.NewInt(1)), ShouldBeZeroValue)
							So(accountInfo.ID, ShouldResemble, likeChainID)
							So(accountInfo.NextNonce, ShouldEqual, 1)
						})
					})
				})
			})
		})

		Convey("Given a TransferTransaction with normal remark", func() {
			outputs := []txs.TransferOutput{
				{
					To:     types.ID(likeChainIDs[1]),
					Value:  types.NewBigInt(1),
					Remark: []byte("99BottlesOfBeer"),
				},
			}
			rawTx := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 1, "22ae9bd318079c1665e1e629de8a807d2e2416ff4ed7feb8412d18fc668711e43a2679a66d406a5440dc3457e043cf37c3c05db3a36943c8ee957705b86ca62d1b")
			Convey("The transfer should succeed", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query tx_state should return success", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "success")
				})
				Convey("Query account_info should return the correct balance and increased nonce", func() {
					likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(99)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)
				})
			})
		})

		Convey("Given a TransferTransaction with 4096 bytes remark", func() {
			zeros := make([]byte, 4096)
			outputs := []txs.TransferOutput{
				{
					To:     types.ID(likeChainIDs[1]),
					Value:  types.NewBigInt(1),
					Remark: zeros,
				},
			}
			rawTx := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 1, "35e83a7c43287339f00f3be9b29382c474b5a22004ee00fbca30130fcc3638e83aa7abeb7476ff9aec88fa4f1b132751085e1b3143f5ce1bd77c9180f038dd391c")
			Convey("The transfer should succeed", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query tx_state should return success", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "success")
				})
				Convey("Query account_info should return the correct balance and increased nonce", func() {
					likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(99)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)
				})
			})
		})

		Convey("Given a TransferTransaction with 4097 remark", func() {
			zeros := make([]byte, 4097)
			outputs := []txs.TransferOutput{
				{
					To:     types.ID(likeChainIDs[1]),
					Value:  types.NewBigInt(1),
					Remark: zeros,
				},
			}
			rawTx := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 1, "5bf9ade931bf926978abf198dc93a57d83c2b674013da534d0e95bac2dee5d0c76862205a29e392ffa2cf7be4673ce8644403fa26b007a05274d12d77e1a98681c")
			Convey("The transfer should fail", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.TransferInvalidFormat.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.TransferInvalidFormat.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query tx_state should return fail", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "fail")
				})
				Convey("Query account_info should return unchanged balance and unchaged nonce", func() {
					likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(100)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
			})
		})

		Convey("Given a TransferTransaction from A to B value 0", func() {
			outputs := []txs.TransferOutput{
				{
					To:    types.ID(likeChainIDs[1]),
					Value: types.NewBigInt(0),
				},
			}
			rawTx := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 1, "e4d40af0ead7b81e53194ebc4438cf3d1f48924e8a2a021d11834531d3a3ab8047002d3009fa031de44aaecf9f5f8260354e86259572aca59a50a7262a0938f11b")
			Convey("The transfer should succeed", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query account_info by LikeChain ID should return the correct balances and nextNonce", func() {
					likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(100)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)

					likeChainIDBase64 = base64.StdEncoding.EncodeToString(likeChainIDs[1])
					queryRes = app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[1])
					So(accountInfo.Balance.Cmp(big.NewInt(200)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("Then query tx_state should return success", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "success")
				})
			})
		})

		Convey("Given a TransferTransaction from A to C value 100", func() {
			outputs := []txs.TransferOutput{
				{
					To:    types.ID(likeChainIDs[2]),
					Value: types.NewBigInt(100),
				},
			}
			rawTx := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 1, "c506480e7e4770a25ecc3b6e96544642a6a459433380f836af11c1d8f78d2d3f5939435d133617175aba879e7c3646c21f1fd6b0f45ae901017f08945d9ab03d1c")
			Convey("The transfer should succeed", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query account_info by LikeChain ID should return the correct balances and nextNonce", func() {
					likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(0)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)

					likeChainIDBase64 = base64.StdEncoding.EncodeToString(likeChainIDs[2])
					queryRes = app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo = query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[2])
					So(accountInfo.Balance.Cmp(big.NewInt(400)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 1)
				})
				Convey("Then query tx_state should return success", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "success")
				})
			})
		})

		Convey("Given a TransferTransaction with value sum more than 100", func() {
			outputs := []txs.TransferOutput{
				{
					To:    types.ID(likeChainIDs[1]),
					Value: types.NewBigInt(50),
				},
				{
					To:    types.ID(likeChainIDs[2]),
					Value: types.NewBigInt(51),
				},
			}
			rawTx := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 1, "f66da55da1b7eeb83aa8aa9ac9ddec944eae03d865e4fa3111e030df354f39f46aad5c84aede960cf150ab0eaf9079c706f1036c48d3e60362da89fac7643eb41b")
			Convey("The transfer should fail", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.TransferNotEnoughBalance.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.TransferNotEnoughBalance.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query tx_state should return fail", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "fail")
				})
				Convey("Query account_info should return unchanged balance and increased nonce", func() {
					likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(100)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)
				})
			})
		})

		Convey("Given 2 TransferTransactions with value sum more than 100", func() {
			outputs := []txs.TransferOutput{
				{
					To:    types.ID(likeChainIDs[1]),
					Value: types.NewBigInt(50),
				},
			}
			rawTx1 := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 1, "84861ac52575b2ef6fce0beed36dfdec212d97f3c9a26567c53891520a70fe1f629857af926fb14211008d9410f0e80596a7e4643c650763d3f072c575083c991c")

			outputs = []txs.TransferOutput{
				{
					To:    types.ID(likeChainIDs[2]),
					Value: types.NewBigInt(51),
				},
			}
			rawTx2 := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 2, "3f2a5a285868740707cfc943635d499b4c1a37faa58021bf2619b10b57f777a9738f38eb0a5ecf30844c4ad8051a4df9dc8d87d3c7036954c401a8e2f7cfe9721b")
			Convey("The first TransferTransactions should succeed", func() {
				checkTxRes := app.CheckTx(rawTx1)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx1)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				Convey("The second TransferTransactions should fail", func() {
					deliverTxRes := app.DeliverTx(rawTx2)
					So(deliverTxRes.Code, ShouldEqual, response.TransferNotEnoughBalance.ToResponseDeliverTx().Code)
					app.EndBlock(abci.RequestEndBlock{
						Height: 2,
					})
					app.Commit()
					Convey("Then query tx_state for the first tx should return success", func() {
						txHash := tmhash.Sum(rawTx1)
						queryRes := app.Query(abci.RequestQuery{
							Path: "tx_state",
							Data: txHash,
						})
						So(queryRes.Code, ShouldEqual, response.Success.Code)
						txStateRes := query.GetTxStateRes(queryRes.Value)
						So(txStateRes, ShouldNotBeNil)
						So(txStateRes.Status, ShouldEqual, "success")
						Convey("Then query tx_state for the second tx should return fail", func() {
							txHash := tmhash.Sum(rawTx2)
							queryRes := app.Query(abci.RequestQuery{
								Path: "tx_state",
								Data: txHash,
							})
							So(queryRes.Code, ShouldEqual, response.Success.Code)
							txStateRes := query.GetTxStateRes(queryRes.Value)
							So(txStateRes, ShouldNotBeNil)
							So(txStateRes.Status, ShouldEqual, "fail")
							Convey("Query account_info should return correct balance and increased nonce", func() {
								likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
								queryRes := app.Query(abci.RequestQuery{
									Path: "account_info",
									Data: []byte(likeChainIDBase64),
								})
								So(queryRes.Code, ShouldEqual, response.Success.Code)
								accountInfo := query.GetAccountInfoRes(queryRes.Value)
								So(accountInfo, ShouldNotBeNil)
								So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
								So(accountInfo.Balance.Cmp(big.NewInt(50)), ShouldBeZeroValue)
								So(accountInfo.NextNonce, ShouldEqual, 3)
							})
						})
					})
				})
			})
		})

		Convey("Given a TransferTransaction with invalid nonce", func() {
			outputs := []txs.TransferOutput{
				{
					To:    types.ID(likeChainIDs[1]),
					Value: types.NewBigInt(1),
				},
			}
			rawTx := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 2, "198fc424abd132d3a3fc414fa9c884b103a360441a86d5d51fbf600817b248004220adba5127bfac45d20e6b416f242b66af2e51e27c8f4e18f3ed3e45bdc9421c")
			Convey("The transfer should fail", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.TransferInvalidNonce.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.TransferInvalidNonce.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query tx_state should return fail", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "fail")
				})
			})
		})

		Convey("Given a TransferTransaction with invalid signature", func() {
			outputs := []txs.TransferOutput{
				{
					To:    types.ID(likeChainIDs[1]),
					Value: types.NewBigInt(1),
				},
			}
			rawTx := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 1, "80a1fd124c4b3f1673ff76295e2660280d48711fb2c81aae78d0a9b2fc521e310f9f2a7e59c266852b9a862e880e2bae91359a86372a307041f9342b9c7715c21b")
			Convey("The transfer should fail", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.TransferInvalidSignature.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.TransferInvalidSignature.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query tx_state should return fail", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "fail")
				})
			})
		})

		Convey("Given a TransferTransaction from A to A value 1", func() {
			outputs := []txs.TransferOutput{
				{
					To:    types.ID(likeChainIDs[0]),
					Value: types.NewBigInt(1),
				},
			}
			rawTx := txs.RawTransferTx(types.ID(likeChainIDs[0]), outputs, types.NewBigInt(0), 1, "11a0ac35b133a0f31c4d330e446581ed0e110e9ee8954a7d9ce8491830e9ae8e0ff0e618c5590f4c95857e0573e5e13985cff95bbcc90c0bb811b629716cab1f1c")
			Convey("The transfer should succeed", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query account_info by LikeChain ID should return the correct balances and nextNonce", func() {
					likeChainIDBase64 := base64.StdEncoding.EncodeToString(likeChainIDs[0])
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.ID, ShouldResemble, likeChainIDs[0])
					So(accountInfo.Balance.Cmp(big.NewInt(100)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)
				})
				Convey("Then query tx_state should return success", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "success")
				})
			})
		})
	})
}

func TestWithdraw(t *testing.T) {
	Convey("Given account A with 100 balance", t, func() {
		mockCtx := context.NewMock()
		app := &LikeChainApplication{
			ctx: mockCtx.ApplicationContext,
		}
		app.InitChain(abci.RequestInitChain{})

		regInfo := struct {
			Addr string
			Sig  string
		}{
			"0x539c17e9e5fd1c8e3b7506f4a7d9ba0a0677eae9",
			"65e6d31224fbcec8e41251d7b014e569d4a94c866227637c6b1fcf75a4505f241b2009557e79d5879a8bfbbb5dec86205c3481ed3042ad87f0643778022f54141b",
		}

		rawTx := txs.RawRegisterTx(regInfo.Addr, regInfo.Sig)
		deliverTxRes := app.DeliverTx(rawTx)
		So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
		likeChainID := deliverTxRes.Data
		account.SaveBalance(mockCtx.GetMutableState(), types.ID(likeChainID), big.NewInt(100))

		app.EndBlock(abci.RequestEndBlock{
			Height: 1,
		})
		app.Commit()

		likeChainIDBase64 := "bDH8FUIuutKKr5CJwwZwL2dUC1M="
		So(types.IDStr(likeChainIDBase64), ShouldResemble, types.ID(likeChainID))

		queryRes := app.Query(abci.RequestQuery{
			Path: "account_info",
			Data: []byte(likeChainIDBase64),
		})
		So(queryRes.Code, ShouldEqual, response.Success.Code)
		accountInfo := query.GetAccountInfoRes(queryRes.Value)
		So(accountInfo, ShouldNotBeNil)
		So(accountInfo.Balance.Cmp(big.NewInt(100)), ShouldBeZeroValue)
		So(accountInfo.NextNonce, ShouldEqual, 1)

		Convey("Given a WithdrawTransaction from A to a certain address with value 1", func() {
			rawTx := txs.RawWithdrawTx(types.ID(likeChainID), "0x833a907efe57af3040039c90f4a59946a0bb3d47", types.NewBigInt(1), types.NewBigInt(0), 1, "d2354ea2e358bfd8e40d7afeaf6dbc79f6241d5517c398b5901f5162b7d9a09e58d2bdaaaf577ed28d1b871fea7a20572f2bf3865d6bad7e82687967c5cb63dd1c")
			Convey("The withdraw should succeed", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				packedTx := deliverTxRes.Data
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query account_info should return the correct balance and nonce", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.Balance.Cmp(big.NewInt(99)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)
				})
				Convey("Then query tx_state should return success", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "success")
				})
				Convey("Then query withdraw_proof should return a proof", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path:   "withdraw_proof",
						Data:   packedTx,
						Height: 2,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					proof := iavl.RangeProof{}
					err := json.Unmarshal(queryRes.Value, &proof)
					So(err, ShouldBeNil)
					Convey("The proof should be corresponding to the withdraw tree hash", func() {
						err := proof.Verify(mockCtx.GetMutableState().GetAppHash()[:20])
						So(err, ShouldBeNil)
					})
				})
				Convey("Then query withdraw_proof with changed info should return fail", func() {
					packedTx[0]++
					queryRes := app.Query(abci.RequestQuery{
						Path:   "withdraw_proof",
						Data:   packedTx,
						Height: 2,
					})
					So(queryRes.Code, ShouldEqual, response.QueryWithdrawProofNotExist.Code)
				})
				Convey("But repeated withdraw with the same transaction should fail", func() {
					checkTxRes := app.CheckTx(rawTx)
					So(checkTxRes.Code, ShouldEqual, response.WithdrawDuplicated.ToResponseCheckTx().Code)
					deliverTxRes := app.DeliverTx(rawTx)
					So(deliverTxRes.Code, ShouldEqual, response.WithdrawDuplicated.ToResponseDeliverTx().Code)
					app.EndBlock(abci.RequestEndBlock{
						Height: 3,
					})
					app.Commit()
					Convey("Then query tx_state should return success", func() {
						txHash := tmhash.Sum(rawTx)
						queryRes := app.Query(abci.RequestQuery{
							Path: "tx_state",
							Data: txHash,
						})
						So(queryRes.Code, ShouldEqual, response.Success.Code)
						txStateRes := query.GetTxStateRes(queryRes.Value)
						So(txStateRes, ShouldNotBeNil)
						So(txStateRes.Status, ShouldEqual, "success")
					})
					Convey("Then query account_info should return the correct balance and nonce", func() {
						queryRes := app.Query(abci.RequestQuery{
							Path: "account_info",
							Data: []byte(likeChainIDBase64),
						})
						So(queryRes.Code, ShouldEqual, response.Success.Code)
						accountInfo := query.GetAccountInfoRes(queryRes.Value)
						So(accountInfo, ShouldNotBeNil)
						So(accountInfo.Balance.Cmp(big.NewInt(99)), ShouldBeZeroValue)
						So(accountInfo.NextNonce, ShouldEqual, 2)
					})
				})
			})
		})

		Convey("Given a WithdrawTransaction from A's Ethereum address to a certain address with value 1", func() {
			rawTx := txs.RawWithdrawTx(types.Addr("0x539c17e9e5fd1c8e3b7506f4a7d9ba0a0677eae9"), "0x833a907efe57af3040039c90f4a59946a0bb3d47", types.NewBigInt(1), types.NewBigInt(0), 1, "cfd63e8ff3991492c7eb56723ec12fdcc2e145b20c0de2a578ce63c268ad770f4f3361e27a8ae34fdf7b897f13a09b2e544eca7a8d533db28af42d54ff4df08d1c")
			Convey("The withdraw should succeed", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				packedTx := deliverTxRes.Data
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query tx_state should return success", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "success")
				})
				Convey("Then query account_info should return the correct balance", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.Balance.Cmp(big.NewInt(99)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)
				})
				Convey("Then query withdraw_proof should return a proof", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path:   "withdraw_proof",
						Data:   packedTx,
						Height: 2,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					proof := iavl.RangeProof{}
					err := json.Unmarshal(queryRes.Value, &proof)
					So(err, ShouldBeNil)
					Convey("The proof should be corresponding to the withdraw tree hash", func() {
						err := proof.Verify(mockCtx.GetMutableState().GetAppHash()[:20])
						So(err, ShouldBeNil)
					})
				})
			})
		})

		Convey("Given a WithdrawTransaction from A to a certain address with value 100", func() {
			rawTx := txs.RawWithdrawTx(types.ID(likeChainID), "0x833a907efe57af3040039c90f4a59946a0bb3d47", types.NewBigInt(100), types.NewBigInt(0), 1, "3b0ea1e2e032d01b559f6d27a92c6be0372fb4d5d54ee6707835b6f217d1fa7226e9d2e1180331dfd12a880639e98bc8aa10349fba1da467cb2784eddfa903d41b")
			Convey("The withdraw should succeed", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.Success.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.Success.ToResponseDeliverTx().Code)
				packedTx := deliverTxRes.Data
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query tx_state should return success", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "success")
				})
				Convey("Then query account_info should return the correct balance", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.Balance.Cmp(big.NewInt(0)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)
				})
				Convey("Then query withdraw_proof should return a proof", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path:   "withdraw_proof",
						Data:   packedTx,
						Height: 2,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					proof := iavl.RangeProof{}
					err := json.Unmarshal(queryRes.Value, &proof)
					So(err, ShouldBeNil)
					Convey("The proof should be corresponding to the withdraw tree hash", func() {
						err := proof.Verify(mockCtx.GetMutableState().GetAppHash()[:20])
						So(err, ShouldBeNil)
					})
				})
			})
		})

		Convey("Given a WithdrawTransaction from A to a certain address with value 101", func() {
			rawTx := txs.RawWithdrawTx(types.ID(likeChainID), "0x833a907efe57af3040039c90f4a59946a0bb3d47", types.NewBigInt(101), types.NewBigInt(0), 1, "d7abbd0ffeca27528cf28816faaf6b9e412f020d1f453250880071a7c3515fea12b1ac8594c7b893946efd723efe62915122e662da261da7336fce90623f7c8e1b")
			Convey("The withdraw should fail", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.WithdrawNotEnoughBalance.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.WithdrawNotEnoughBalance.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query tx_state should return fail", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "fail")
				})
				Convey("Then query account_info should return unchanged balance and increased nonce", func() {
					queryRes := app.Query(abci.RequestQuery{
						Path: "account_info",
						Data: []byte(likeChainIDBase64),
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					accountInfo := query.GetAccountInfoRes(queryRes.Value)
					So(accountInfo, ShouldNotBeNil)
					So(accountInfo.Balance.Cmp(big.NewInt(100)), ShouldBeZeroValue)
					So(accountInfo.NextNonce, ShouldEqual, 2)
				})
			})
		})

		Convey("Given a WithdrawTransaction with invalid signature", func() {
			rawTx := txs.RawWithdrawTx(types.ID(likeChainID), "0x833a907efe57af3040039c90f4a59946a0bb3d47", types.NewBigInt(101), types.NewBigInt(0), 1, "e828d630862be9e3564d0723c875ea93b1ec6be17c42f2a7345909d55f0b403024a1471b1000339e2a9f026d8e47d9f0afa856f899e671328b0fe63436e555911c")
			Convey("The withdraw should fail with InvalidSignature", func() {
				checkTxRes := app.CheckTx(rawTx)
				So(checkTxRes.Code, ShouldEqual, response.WithdrawInvalidSignature.ToResponseCheckTx().Code)
				deliverTxRes := app.DeliverTx(rawTx)
				So(deliverTxRes.Code, ShouldEqual, response.WithdrawInvalidSignature.ToResponseDeliverTx().Code)
				app.EndBlock(abci.RequestEndBlock{
					Height: 2,
				})
				app.Commit()
				Convey("Then query tx_state should return fail", func() {
					txHash := tmhash.Sum(rawTx)
					queryRes := app.Query(abci.RequestQuery{
						Path: "tx_state",
						Data: txHash,
					})
					So(queryRes.Code, ShouldEqual, response.Success.Code)
					txStateRes := query.GetTxStateRes(queryRes.Value)
					So(txStateRes, ShouldNotBeNil)
					So(txStateRes.Status, ShouldEqual, "fail")
					Convey("Then query account_info should return unchanged balance and nonce", func() {
						queryRes := app.Query(abci.RequestQuery{
							Path: "account_info",
							Data: []byte(likeChainIDBase64),
						})
						So(queryRes.Code, ShouldEqual, response.Success.Code)
						accountInfo := query.GetAccountInfoRes(queryRes.Value)
						So(accountInfo, ShouldNotBeNil)
						So(accountInfo.Balance.Cmp(big.NewInt(100)), ShouldBeZeroValue)
						So(accountInfo.NextNonce, ShouldEqual, 1)
					})
				})
			})
		})
	})
}

func TestGC(t *testing.T) {
	Convey("Given an application state set with keep_blocks = 10", t, func() {
		appConf.GetConfig().KeepBlocks = 10

		mockCtx := context.NewMock()
		app := &LikeChainApplication{
			ctx: mockCtx.ApplicationContext,
		}
		app.InitChain(abci.RequestInitChain{})
		app.Commit()
		initialStateTreeVersion := mockCtx.ApplicationContext.GetMutableState().MutableStateTree().Version64()
		initialWithdrawTreeVersion := mockCtx.ApplicationContext.GetMutableState().MutableWithdrawTree().Version64()

		app.Commit()
		secondStateTreeVersion := mockCtx.ApplicationContext.GetMutableState().MutableStateTree().Version64()
		secondWithdrawTreeVersion := mockCtx.ApplicationContext.GetMutableState().MutableWithdrawTree().Version64()

		Convey("After committing 9 blocks", func() {
			for i := 0; i < 8; i++ {
				app.Commit()
			}
			Convey("Should still be able to get the initial versions", func() {
				stateTree := mockCtx.ApplicationContext.GetMutableState().MutableStateTree()
				So(stateTree.VersionExists(initialStateTreeVersion), ShouldBeTrue)
				withdrawTree := mockCtx.ApplicationContext.GetMutableState().MutableWithdrawTree()
				So(withdrawTree.VersionExists(initialWithdrawTreeVersion), ShouldBeTrue)
				Convey("After committing 10 blocks", func() {
					app.Commit()
					Convey("The initial versions should be GCed", func() {
						stateTree := mockCtx.ApplicationContext.GetMutableState().MutableStateTree()
						So(stateTree.VersionExists(initialStateTreeVersion), ShouldBeFalse)
						withdrawTree := mockCtx.ApplicationContext.GetMutableState().MutableWithdrawTree()
						So(withdrawTree.VersionExists(initialWithdrawTreeVersion), ShouldBeFalse)
						Convey("The second versions should still be there", func() {
							So(stateTree.VersionExists(secondStateTreeVersion), ShouldBeTrue)
							So(withdrawTree.VersionExists(secondWithdrawTreeVersion), ShouldBeTrue)
						})
					})
				})
			})
		})
	})
}
