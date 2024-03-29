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

// GetNetInfo to get no.of peers and addresses
func GetNetInfo(ops HTTPOptions, cfg *config.Config, c client.Client) {
	bp, err := createBatchPoints(cfg.InfluxDB.Database)
	if err != nil {
		return
	}
	var pts []*client.Point

	resp, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error getting node_info: %v", err)
		return
	}
	var ni NetInfo
	err = json.Unmarshal(resp.Body, &ni)
	if err != nil {
		log.Printf("Error while unamrshelling NetInfo: %v", err)
		return
	}

	_, synced := GetNodeSync(cfg, c) // get syncing status

	numPeers, err := strconv.Atoi(ni.Result.NumPeers)
	if err != nil {
		log.Printf("Error converting num_peers to int: %v", err)
		numPeers = 0
	} else if strings.ToUpper(cfg.PeersAlert.EnableAlert) == "YES" && int64(numPeers) < cfg.PeersAlert.NumPeersThreshold {
		if synced == "1" {
			_ = SendTelegramAlert(fmt.Sprintf("Number of peers connected to your validator has fallen below %d", cfg.PeersAlert.NumPeersThreshold), cfg)
			_ = SendEmailAlert(fmt.Sprintf("Number of peers connected to your validator has fallen below %d", cfg.PeersAlert.NumPeersThreshold), cfg)
		}
	}
	p1, err := createDataPoint("vab_num_peers", map[string]string{}, map[string]interface{}{"count": numPeers})
	if err == nil {
		pts = append(pts, p1)
	}

	bp.AddPoints(pts)
	_ = writeBatchPoints(c, bp)
	log.Printf("No. of peers: %d \n", numPeers)
}

// GetPeersCount returns count of peer addresses from db
func GetPeersCount(cfg *config.Config, c client.Client) string {
	var count string
	q := client.NewQuery("SELECT last(count) FROM vab_num_peers", cfg.InfluxDB.Database, "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		for _, r := range response.Results {
			if len(r.Series) != 0 {
				for idx, col := range r.Series[0].Columns {
					if col == "last" {
						c := r.Series[0].Values[0][idx]
						count = fmt.Sprintf("%v", c)
						break
					}
				}
			}
		}
	}

	return count
}
