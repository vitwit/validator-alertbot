package targets

import (
	"encoding/json"
	"fmt"
	"log"

	"strconv"

	"validator-alertbot/config"

	client "github.com/influxdata/influxdb1-client/v2"
)

func MissedBlocks(ops HTTPOptions, cfg *config.Config, c client.Client) {
	var prev_missed_counter int64
	bp, err := createBatchPoints(cfg.InfluxDB.Database)
	if err != nil {
		return
	}
	resp, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("⛔⛔ Error while making a connection: %v", err)
		return
	}

	var b MissedBlock

	err = json.Unmarshal(resp.Body, &b)

	if err != nil {
		log.Printf("Error while unmarshelling data : %v", err)
		return
	}

	a := cfg.MissedBlocksAlert.MissedBlocksThreshold
	temp := b.ValSigningInfo.MissedBlocksCounter

	i, err := strconv.ParseInt(temp, 10, 64)
	if err != nil {
		fmt.Println("error while converting string to int")
	}
	q := client.NewQuery("SELECT last(missed_block_counter) FROM val_missed_blocks", cfg.InfluxDB.Database, "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {

		for _, r := range response.Results {
			if len(r.Series) != 0 {
				for idx, col := range r.Series[0].Columns {
					if col == "last" {
						temp1 := r.Series[0].Values[0][idx]
						a := fmt.Sprintf("%v", temp1)
						prev_missed_counter, _ = strconv.ParseInt(a, 10, 64)

					}
				}
			}
		}

	}

	_ = writeToInfluxDb(c, bp, "val_missed_blocks", map[string]string{}, map[string]interface{}{"missed_block_counter": temp})

	if i-prev_missed_counter >= a {
		_ = SendTelegramAlert(fmt.Sprintf("%s validator missed more than the threshold provided", cfg.ValidatorName), cfg)
		_ = SendEmailAlert(fmt.Sprintf("%s validator missed more than the threshold provided", cfg.ValidatorName), cfg)

	}

}

func IndexOffSet(ops HTTPOptions, cfg *config.Config, c client.Client) {
	var prev_index_offset int64
	bp, err := createBatchPoints(cfg.InfluxDB.Database)
	if err != nil {
		return
	}
	resp, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("⛔⛔ Error while making a connection: %v", err)
		return
	}
	var b MissedBlock

	err = json.Unmarshal(resp.Body, &b)

	if err != nil {
		log.Printf("Error while unmarshelling data : %v", err)
		return
	}
	a := cfg.MissedBlocksAlert.IndexOffSetThreshold
	temp := b.ValSigningInfo.IndexOffset
	i, err := strconv.ParseInt(temp, 10, 64)
	if err != nil {
		fmt.Println("error while converting string to int")
	}
	q := client.NewQuery("SELECT last(index_offset) FROM val_index_offset", cfg.InfluxDB.Database, "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {

		for _, r := range response.Results {
			if len(r.Series) != 0 {
				for idx, col := range r.Series[0].Columns {
					if col == "last" {
						temp1 := r.Series[0].Values[0][idx]

						a := fmt.Sprintf("%v", temp1)

						prev_index_offset, _ = strconv.ParseInt(a, 10, 64)

					}
				}
			}
		}

	}

	_ = writeToInfluxDb(c, bp, "val_index_offset", map[string]string{}, map[string]interface{}{"index_offset": temp})

	if i-prev_index_offset < a {
		_ = SendTelegramAlert(fmt.Sprintf("%s validator index offset is less than the threshold", cfg.ValidatorName), cfg)
		_ = SendEmailAlert(fmt.Sprintf("%s validator index offset is less than the threshold", cfg.ValidatorName), cfg)

	}
}
