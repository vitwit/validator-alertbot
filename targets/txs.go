package targets

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"validator-alertbot/config"

	client "github.com/influxdata/influxdb1-client/v2"
)

// TxAlerts
func TxAlerts(ops HTTPOptions, cfg *config.Config, c client.Client) {
	resp, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error in get tx: %v", err)
		return
	}

	var networkLatestBlock NetworkLatestBlock
	err = json.Unmarshal(resp.Body, &networkLatestBlock)
	if err != nil {
		log.Printf("Error while unmarshelling NetworkLatestBlock: %v", err)
		return
	}

	cbh := networkLatestBlock.Result.SyncInfo.LatestBlockHeight

	resp, err = HitHTTPTarget(HTTPOptions{
		Endpoint:    cfg.ExternalRPC + "/block",
		QueryParams: QueryParams{"height": cbh},
		Method:      "GET",
	})
	if err != nil {
		log.Printf("Error getting details of current block: %v", err)
		return
	}

	var b BlockResponse
	err = json.Unmarshal(resp.Body, &b)
	if err != nil {
		log.Printf("Error while unmarshelling BlockResponse: %v", err)
		return
	}

	txs := b.Result.Block.Data.Txs
	for _, t := range txs {
		txHash := GenerateHash(t)
		log.Printf("tx hash.. : %s", txHash)
		resp, err = HitHTTPTarget(HTTPOptions{
			Endpoint: cfg.LCDEndpoint + "/cosmos/tx/v1beta1/txs/" + txHash,
			Method:   "GET",
		})
		if err != nil {
			log.Printf("Error in transactoons : %v", err)
			return
		}

		if &resp.Body == nil {
			log.Printf("Error while getting txs : %v", err)
			return
		}

		var tx TxHashResp
		err = json.Unmarshal(resp.Body, &tx)
		if err != nil {
			log.Printf("Error while unmarshelling TxHashResp: %v", err)
			return
		}

		var msgIndex = 1

		if len(tx.TxResponse.Logs) != 0 {
			msgIndex = tx.TxResponse.Logs[0].MsgIndex // check tx status, If it's 0 then valid otherwise not
		}

		log.Printf("Tx Status : %d", msgIndex)

		if msgIndex == 0 {
			if len(tx.TxResponse.Tx.Body.Messages) != 0 {
				txType := tx.TxResponse.Tx.Body.Messages[0].Type
				txValue := tx.TxResponse.Tx.Body.Messages[0]
				log.Printf("txType: %s", txType)

				// Calling function to get voting power change alert msg
				votingPowerMsg := GetValidatorVotingPower(ops, cfg, c)

				if txType == "/cosmos.staking.v1beta1.MsgDelegate" {

					amount := txValue.Amount.Amount
					amountInAKT := ConvertToAKT(amount)
					delegatorAddress := txValue.DelegatorAddress

					bal := ConvertToFolat64(amount)
					if bal > cfg.DelegationAlerts.DelegationAmountThreshold {
						if txValue.ValidatorAddress == cfg.ValOperatorAddress {
							_ = SendTelegramAlert(fmt.Sprintf("Delegation alert: you have a new delegation %s from %s and %s ", amountInAKT, delegatorAddress, votingPowerMsg), cfg)
							_ = SendEmailAlert(fmt.Sprintf("Delegation alert: you have a new delegation %s from %s and %s", amountInAKT, delegatorAddress, votingPowerMsg), cfg)
						}
					}
				} else if txType == "/cosmos.staking.v1beta1.MsgUndelegate" {
					amount := txValue.Amount.Amount
					amountInAKT := ConvertToAKT(amount)

					if txValue.DelegatorAddress == cfg.AccountAddress || txValue.ValidatorAddress == cfg.ValOperatorAddress {

						_ = SendTelegramAlert(fmt.Sprintf("Undelegation alert: Undelegated %s from your validator. %s", amountInAKT, votingPowerMsg), cfg)
						_ = SendEmailAlert(fmt.Sprintf("Undelegation alert: Undelegated %s from your validator. %s", amountInAKT, votingPowerMsg), cfg)
					}
				} else if txType == "/cosmos.staking.v1beta1.MsgBeginRedelegate" {
					amount := txValue.Amount.Amount
					amountInAKT := ConvertToAKT(amount)

					if txValue.ValidatorSrcAddress == cfg.ValOperatorAddress {
						_ = SendTelegramAlert(fmt.Sprintf("Redelegation alert: Redelegated %s from your validator to %s. %s", amountInAKT, txValue.ValidatorDstAddress, votingPowerMsg), cfg)
						_ = SendEmailAlert(fmt.Sprintf("Redelegation alert: Redelegated %s from validator. %s ", amountInAKT, votingPowerMsg), cfg)
					}

					if txValue.ValidatorDstAddress == cfg.ValOperatorAddress {
						_ = SendTelegramAlert(fmt.Sprintf("Redelegation alert: Redelegated %s to your validator from %s. %s", amountInAKT, txValue.ValidatorSrcAddress, votingPowerMsg), cfg)
						_ = SendEmailAlert(fmt.Sprintf("Redelegation alert: Redelegated %s to validator. %s ", amountInAKT, votingPowerMsg), cfg)
					}
				}
			}
		} else {
			return
		}
	}
}

//GenerateHash returns hash of a transaction
func GenerateHash(Txbytes string) string {
	txBytes, err := base64.StdEncoding.DecodeString(Txbytes)
	if err != nil {
		fmt.Println("unable to decode string", err.Error())
	}

	hash := sha256.New()
	_, err = hash.Write(txBytes)

	return fmt.Sprintf("%x", hash.Sum(nil))
}
