package targets

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"validator-alertbot/config"

	client "github.com/influxdata/influxdb1-client/v2"
)

// GetRewradsAndCommission is to get current rewards and commission of a validator using operator address
func GetRewradsAndCommission(ops HTTPOptions, cfg *config.Config, c client.Client) {
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

	var rewardsResp DistributionRewards
	err = json.Unmarshal(resp.Body, &rewardsResp)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	var commission, rewards string

	if len(rewardsResp.Result.SelfBondRewards) > 0 {
		rewards = rewardsResp.Result.SelfBondRewards[0].Amount
		log.Printf("Val Rewards: %s", rewards)
	}

	if len(rewardsResp.Result.ValCommission) > 0 {
		commission = rewardsResp.Result.ValCommission[0].Amount
		log.Printf("Val Commission: %s", rewards)
	}

	if commission != "" && rewards != "" {
		com, _ := strconv.ParseFloat(commission, 64)
		r, _ := strconv.ParseFloat(rewards, 64)

		total := com + r
		s := fmt.Sprintf("%f", total)
		totalRewrads := ConvertToAKT(s)

		_ = writeToInfluxDb(c, bp, "vab_total_rewards", map[string]string{}, map[string]interface{}{"rewards": totalRewrads})
		log.Printf("Validator total Rewrads: %s", totalRewrads)
	}
}

// GetRewardsFromDB returns the validator rewards from db
func GetRewardsFromDB(cfg *config.Config, c client.Client) string {
	var rewards string
	q := client.NewQuery("SELECT last(rewards) FROM vab_total_rewards", cfg.InfluxDB.Database, "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		for _, r := range response.Results {
			if len(r.Series) != 0 {
				for idx, col := range r.Series[0].Columns {
					if col == "last" {
						status := r.Series[0].Values[0][idx]
						rewards = fmt.Sprintf("%v", status)
						break
					}
				}
			}
		}
	}
	return rewards
}
