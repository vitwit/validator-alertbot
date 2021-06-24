package targets

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"validator-alertbot/config"

	client "github.com/influxdata/influxdb1-client/v2"
)

// SendSingleMissedBlockAlert to send missed block alerting
func SendSingleMissedBlockAlert(cfg *config.Config, c client.Client, addrExists bool, cbh string) error {
	bp, err := createBatchPoints(cfg.InfluxDB.Database)
	if err != nil {
		return err
	}

	if !addrExists {
		_, synced := GetNodeSync(cfg, c) // get syncing status

		if cfg.MissedBlocksAlert.MissedBlocksThreshold == 1 {
			// Calling function to get validator latest block height
			if synced == "1" {
				if strings.ToUpper(cfg.MissedBlocksAlert.EnableAlert) == "YES" {
					_ = SendTelegramAlert(fmt.Sprintf("%s validator missed a block at block height %s", cfg.ValidatorName, cbh), cfg)
					_ = SendEmailAlert(fmt.Sprintf("%s validator missed a block at block height %s", cfg.ValidatorName, cbh), cfg)
					log.Println("Sent missed block alerting")
				}
			}
			_ = writeToInfluxDb(c, bp, "vab_missed_blocks", map[string]string{}, map[string]interface{}{"block_height": cbh, "current_height": cbh})
		}
	}

	// Calling function to check validator jailed status
	err = CheckValidatorJailed(cfg)
	if err != nil {
		log.Printf("Error while sending jailed alerting: %v", err)
		return err
	}

	return nil
}

// MissedBlocks sends alerts of missed blocks according to the threshold given by user
func MissedBlocks(ops HTTPOptions, cfg *config.Config, c client.Client) {
	bp, err := createBatchPoints(cfg.InfluxDB.Database)
	if err != nil {
		return
	}

	resp, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error in missed blocks: %v", err)
		// _ = SendTelegramAlert(fmt.Sprintf("⛔⛔ Unreachable to EXTERNAL RPC :: %s and the ERROR is : %v", ops.Endpoint, err.Error()), cfg)
		return
	}

	var networkLatestBlock NetworkLatestBlock
	err = json.Unmarshal(resp.Body, &networkLatestBlock)
	if err != nil {
		log.Printf("Error while unmarshelling NetworkLatestBlock : %v", err)
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

	if &b != nil {
		addrExists := false
		for _, c := range b.Result.Block.LastCommit.Signatures {
			if strings.EqualFold(c.ValidatorAddress, cfg.ValidatorHexAddress) {
				addrExists = true
			}
		}

		log.Println("address exists and height......", addrExists, cbh)

		if !addrExists {

			blocks := GetContinuousMissedBlock(cfg, c)
			currentHeightFromDb := GetlatestCurrentHeightFromDB(cfg, c)
			blocksArray := strings.Split(blocks, ",")
			fmt.Println("blocks length ", int64(len(blocksArray)), currentHeightFromDb)
			// calling function to store single blocks
			err = SendSingleMissedBlockAlert(cfg, c, addrExists, cbh)
			if err != nil {
				log.Printf("Error while sending missed block alert: %v", err)

			}
			if strings.ToUpper(cfg.MissedBlocksAlert.EnableAlert) == "YES" && cfg.MissedBlocksAlert.MissedBlocksThreshold > 1 {
				_, synced := GetNodeSync(cfg, c) // get syncing status

				if int64(len(blocksArray))-1 >= cfg.MissedBlocksAlert.MissedBlocksThreshold {
					missedBlocks := strings.Split(blocks, ",")
					if synced == "1" {
						err = SendTelegramAlert(fmt.Sprintf("%s validator missed blocks from height %s to %s", cfg.ValidatorName, missedBlocks[0], missedBlocks[len(missedBlocks)-2]), cfg)
						err = SendEmailAlert(fmt.Sprintf("%s validator missed blocks from height %s to %s", cfg.ValidatorName, missedBlocks[0], missedBlocks[len(missedBlocks)-2]), cfg)
					}
					err = writeToInfluxDb(c, bp, "vab_missed_blocks", map[string]string{}, map[string]interface{}{"block_height": "", "current_height": cbh})
					if err != nil {
						return
					}
					return

				}
				if len(blocksArray) == 1 {
					blocks = cbh + ","
				} else {
					rpcBlockHeight, _ := strconv.Atoi(cbh)
					dbBlockHeight, _ := strconv.Atoi(currentHeightFromDb)
					diff := rpcBlockHeight - dbBlockHeight
					if diff == 1 {
						blocks = blocks + cbh + ","
					} else if diff > 1 {
						blocks = ""
					}
				}
				err = writeToInfluxDb(c, bp, "vab_missed_blocks", map[string]string{}, map[string]interface{}{"block_height": blocks, "current_height": cbh})
				if err != nil {
					return
				}
				return

			}
		}
	} else {
		log.Println("Got an empty response from external rpc block dataa...")
	}

	return
}

// GetContinuousMissedBlock returns the latest missed block from the db
func GetContinuousMissedBlock(cfg *config.Config, c client.Client) string {
	var blocks string
	q := client.NewQuery("SELECT last(block_height) FROM vab_missed_blocks", cfg.InfluxDB.Database, "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		for _, r := range response.Results {
			if len(r.Series) != 0 {
				for idx, col := range r.Series[0].Columns {
					if col == "last" {
						heightValue := r.Series[0].Values[0][idx]
						blocks = fmt.Sprintf("%v", heightValue)
						break
					}
				}
			}
		}
	}
	return blocks
}

// GetlatestCurrentHeightFromDB returns latest current height from db
func GetlatestCurrentHeightFromDB(cfg *config.Config, c client.Client) string {
	var currentHeight string
	q := client.NewQuery("SELECT last(current_height) FROM vab_missed_blocks", cfg.InfluxDB.Database, "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		for _, r := range response.Results {
			if len(r.Series) != 0 {
				for idx, col := range r.Series[0].Columns {
					if col == "last" {
						heightValue := r.Series[0].Values[0][idx]
						currentHeight = fmt.Sprintf("%v", heightValue)
						break
					}
				}
			}
		}
	}
	return currentHeight
}
