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

// GetNetworkLatestBlock to get latest block height of a network
func GetNetworkLatestBlock(ops HTTPOptions, cfg *config.Config, c client.Client) {
	bp, err := createBatchPoints(cfg.InfluxDB.Database)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	resp, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	var networkBlock NetworkLatestBlock
	err = json.Unmarshal(resp.Body, &networkBlock)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if &networkBlock != nil {

		networkBlockHeight, err := strconv.Atoi(networkBlock.Result.SyncInfo.LatestBlockHeight)
		if err != nil {
			log.Println("Error while converting network height from string to int ", err)
		}
		_ = writeToInfluxDb(c, bp, "vab_network_latest_block", map[string]string{}, map[string]interface{}{"block_height": networkBlockHeight})
		log.Printf("Network height: %d", networkBlockHeight)

		// Calling function to get validator latest
		// block height
		validatorHeight := GetValidatorBlockHeight(cfg, c)
		if validatorHeight == "" {
			log.Println("Error while fetching validator block height from db ", validatorHeight)
			return
		}

		vaidatorBlockHeight, _ := strconv.Atoi(validatorHeight)
		heightDiff := networkBlockHeight - vaidatorBlockHeight

		_ = writeToInfluxDb(c, bp, "vab_height_difference", map[string]string{}, map[string]interface{}{"difference": heightDiff})
		log.Printf("Network height: %d and Validator Height: %d", networkBlockHeight, vaidatorBlockHeight)

		blockDiffThreshold := cfg.BlockDiffAlert.BlockDiffThreshold
		// Send alert
		if strings.ToUpper(cfg.BlockDiffAlert.EnableAlert) == "YES" && int64(heightDiff) >= blockDiffThreshold {
			_ = SendTelegramAlert(fmt.Sprintf("Block difference between network and validator has exceeded %d", blockDiffThreshold), cfg)
			_ = SendEmailAlert(fmt.Sprintf("Block difference between network and validator has exceeded %d", blockDiffThreshold), cfg)

			log.Println("Sent alert of block height difference")
		}
	}
}

// GetNetworkBlock returns network current block height
func GetNetworkBlock(cfg *config.Config, c client.Client) string {
	var networkHeight string
	q := client.NewQuery("SELECT last(block_height) FROM vab_network_latest_block", cfg.InfluxDB.Database, "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		for _, r := range response.Results {
			if len(r.Series) != 0 {
				for idx, col := range r.Series[0].Columns {
					if col == "last" {
						heightValue := r.Series[0].Values[0][idx]
						networkHeight = fmt.Sprintf("%v", heightValue)
						break
					}
				}
			}
		}
	}

	return networkHeight
}