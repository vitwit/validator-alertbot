package targets

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"validator-alertbot/config"

	client "github.com/influxdata/influxdb1-client/v2"
)

// GetValidatorVotingPower to get voting power of a validator
func GetValidatorVotingPower(ops HTTPOptions, cfg *config.Config, c client.Client) {
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

	var validatorResp ValidatorResp
	err = json.Unmarshal(resp.Body, &validatorResp)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	vp := validatorResp.Result.DelegatorShares
	vp1 := convertValue(vp)

	votingPowerFromDb := GetVotingPowerFromDb(cfg, c)

	if vp1 != votingPowerFromDb {
		if strings.ToUpper(cfg.VotingPowerAlert.EnableAlert) == "YES" {
			_ = SendTelegramAlert(fmt.Sprintf("Your validator %s voting power has changed from %s to %s", cfg.ValidatorName, votingPowerFromDb, vp1), cfg)
			_ = SendEmailAlert(fmt.Sprintf("Your validator %s voting power has changed from %s to %s", cfg.ValidatorName, votingPowerFromDb, vp1), cfg)
		}
	}

	_ = writeToInfluxDb(c, bp, "vab_voting_power", map[string]string{}, map[string]interface{}{"power": vp1})
	log.Println("Voting Power \n", vp)
}

func convertValue(value string) string {
	bal, _ := strconv.ParseFloat(value, 64)

	a1 := bal / math.Pow(10, 6)
	amount := strconv.FormatFloat(a1, 'f', 0, 64)

	return amount
}

// GetVotingPowerFromDb returns voting power of a validator from db
func GetVotingPowerFromDb(cfg *config.Config, c client.Client) string {
	var vp string
	q := client.NewQuery("SELECT last(power) FROM vab_voting_power", cfg.InfluxDB.Database, "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		for _, r := range response.Results {
			if len(r.Series) != 0 {
				for idx, col := range r.Series[0].Columns {
					if col == "last" {
						v := r.Series[0].Values[0][idx]
						vp = fmt.Sprintf("%v", v)
						break
					}
				}
			}
		}
	}
	return vp
}
