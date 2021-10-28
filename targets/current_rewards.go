package targets

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"validator-alertbot/config"
	"validator-alertbot/utils"

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
		log.Printf("Error in get rewards and commission: %v", err)
		return
	}

	var rewardsResp Rewards
	err = json.Unmarshal(resp.Body, &rewardsResp)
	if err != nil {
		log.Printf("Error in DistributionRewards: %v", err)
		return
	}

	var rewards float64

	if len(rewardsResp.Rewards) > 0 {
		f, _ := strconv.ParseFloat(rewardsResp.Rewards[0].Amount, 64)
		rewards = f
	}

	if rewards != 0 {
		s := fmt.Sprintf("%f", rewards)
		totalRewrads := utils.ConvertToAKT(s, cfg.Denom)

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

// GetValCommission which return the commission of a validator
func GetValCommission(ops HTTPOptions, cfg *config.Config) float64 {
	ops = HTTPOptions{
		Endpoint: cfg.LCDEndpoint + "/cosmos/distribution/v1beta1/validators/" + cfg.ValOperatorAddress + "/commission",
		Method:   http.MethodGet,
	}

	var commission float64

	resp, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error: %v", err)
		return commission
	}

	var result Commission
	err = json.Unmarshal(resp.Body, &result)
	if err != nil {
		log.Printf("Error while unmarshalling commission: %v", err)
		return commission
	}

	if len(result.Commission.Commission) > 0 {
		f, _ := strconv.ParseFloat(result.Commission.Commission[0].Amount, 64)
		commission = f

	}

	return commission
}
