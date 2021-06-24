package targets

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"validator-alertbot/config"

	client "github.com/influxdata/influxdb1-client/v2"
)

// GetValStatus to reponse of validator status like
// current block height and node status
func GetValStatus(cfg *config.Config, c client.Client) (string, int) {
	bp, err := createBatchPoints(cfg.InfluxDB.Database)
	if err != nil {
		return "", -1
	}
	var pts []*client.Point

	resp, err := HitHTTPTarget(HTTPOptions{
		Endpoint: cfg.ValidatorRPCEndpoint + "/status?",
		Method:   http.MethodGet,
	})

	if err != nil {
		log.Printf("Error in get val status: %v", err)
		return "", -1
	}
	var synced int
	var status ValidatorRPCStatus
	err = json.Unmarshal(resp.Body, &status)
	if err != nil {
		log.Printf("Error while unmarshelling ValidatorRPCStatus : %v", err)
		return "", -1
	}

	currentBlockHeight := status.Result.SyncInfo.LatestBlockHeight
	if &status != nil {

		var bh int
		if currentBlockHeight != "" {
			bh, _ = strconv.Atoi(currentBlockHeight)
			p2, err := createDataPoint("vab_current_block_height", map[string]string{}, map[string]interface{}{"height": bh})
			if err == nil {
				pts = append(pts, p2)
			}
		}

		caughtUp := !status.Result.SyncInfo.CatchingUp
		if !caughtUp {
			_ = SendTelegramAlert("Your validator node is not synced!", cfg)
			_ = SendEmailAlert("Your validator node is not synced!", cfg)
			synced = 0
		} else {
			synced = 1
		}
		p3, err := createDataPoint("vab_node_synced", map[string]string{}, map[string]interface{}{"status": synced})
		if err == nil {
			pts = append(pts, p3)
		}

		bp.AddPoints(pts)
		_ = writeBatchPoints(c, bp)
		log.Printf("\nCurrent Block Height: %s \nCaught Up? %t \n",
			currentBlockHeight, caughtUp)
	} else {
		_ = SendTelegramAlert("Validator RPC is not workng!", cfg)
		_ = SendEmailAlert("Validator RPC is not working!", cfg)
	}

	return currentBlockHeight, synced
}

// GetValidatorBlockHeight returns validator current block height from db
func GetValidatorBlockHeight(cfg *config.Config, c client.Client) string {
	var validatorHeight string
	q := client.NewQuery("SELECT last(height) FROM vab_current_block_height", cfg.InfluxDB.Database, "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		for _, r := range response.Results {
			if len(r.Series) != 0 {
				for idx, col := range r.Series[0].Columns {
					if col == "last" {
						heightValue := r.Series[0].Values[0][idx]
						validatorHeight = fmt.Sprintf("%v", heightValue)
						break
					}
				}
			}
		}
	}
	return validatorHeight
}

// GetNodeSync returns the syncing status of a node
func GetNodeSync(cfg *config.Config, c client.Client) (string, string) {
	var status, sync string
	q := client.NewQuery("SELECT last(status) FROM vab_node_synced", cfg.InfluxDB.Database, "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		for _, r := range response.Results {
			if len(r.Series) != 0 {
				for idx, col := range r.Series[0].Columns {
					if col == "last" {
						s := r.Series[0].Values[0][idx]
						sync = fmt.Sprintf("%v", s)
						break
					}
				}
			}
		}
	}

	if sync == "1" {
		status = "synced"
	} else {
		status = "not synced"
	}

	return status, sync
}
