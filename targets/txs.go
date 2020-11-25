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
		log.Printf("Error: %v", err)
		return
	}

	var networkLatestBlock NetworkLatestBlock
	err = json.Unmarshal(resp.Body, &networkLatestBlock)
	if err != nil {
		log.Printf("Error: %v", err)
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
		log.Printf("Error: %v", err)
		return
	}

	txs := b.Result.Block.Data.Txs
	for _, t := range txs {
		txHash := GenerateHash(t)
		resp, err = HitHTTPTarget(HTTPOptions{
			Endpoint: cfg.LCDEndpoint + "/txs/" + txHash,
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
			log.Printf("Error: %v", err)
			return
		}
		if len(tx.Tx.Value.Msg) != 0 {
			txType := tx.Tx.Value.Msg[0].Type
			txValue := tx.Tx.Value.Msg[0].Value
			log.Println("txType: ", txType)

			// Calling function to get voting power change alert msg
			votingPowerMsg := GetValidatorVotingPower(ops, cfg, c)

			if txType == "cosmos-sdk/MsgDelegate" {
				amount := txValue.Amount.Amount
				amountInAKT := ConvertToAKT(amount)
				delegatorAddress := txValue.DelegatorAddress

				if txValue.ValidatorAddress == cfg.ValOperatorAddress {
					_ = SendTelegramAlert(fmt.Sprintf("Delegation alert: you have a new delegation %s from %s and %s ", amountInAKT, delegatorAddress, votingPowerMsg), cfg)
					_ = SendEmailAlert(fmt.Sprintf("Delegation alert: you have a new delegation %s from %s and %s", amountInAKT, delegatorAddress, votingPowerMsg), cfg)
				}
			} else if txType == "cosmos-sdk/MsgUndelegate" {
				amount := txValue.Amount.Amount
				amountInAKT := ConvertToAKT(amount)

				if txValue.DelegatorAddress == cfg.AccountAddress || txValue.ValidatorAddress == cfg.ValOperatorAddress {

					_ = SendTelegramAlert(fmt.Sprintf("Undelegation alert: Undelegated %s from your validator. %s", amountInAKT, votingPowerMsg), cfg)
					_ = SendEmailAlert(fmt.Sprintf("Undelegation alert: Undelegated %s from your validator. %s", amountInAKT, votingPowerMsg), cfg)
				}
			} else if txType == "cosmos-sdk/MsgBeginRedelegate" {
				amount := txValue.Amount.Amount
				amountInAKT := ConvertToAKT(amount)

				if txValue.ValidatorSrcAddress == cfg.ValOperatorAddress {
					_ = SendTelegramAlert(fmt.Sprintf("Reelegation alert: Redelegated %s from validator. %s", amountInAKT, votingPowerMsg), cfg)
					_ = SendEmailAlert(fmt.Sprintf("Redelegation alert: Redelegated %s from validator. %s ", amountInAKT, votingPowerMsg), cfg)
				}

				if txValue.ValidatorDstAddress == cfg.ValOperatorAddress {
					_ = SendTelegramAlert(fmt.Sprintf("Reelegation alert: Redelegated %s to validator. %s", amountInAKT, votingPowerMsg), cfg)
					_ = SendEmailAlert(fmt.Sprintf("Redelegation alert: Redelegated %s to validator. %s ", amountInAKT, votingPowerMsg), cfg)
				}
			}
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
