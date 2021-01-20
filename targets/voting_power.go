package targets

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"validator-alertbot/config"

	client "github.com/influxdata/influxdb1-client/v2"
)

// GetValidatorVotingPower to get voting power of a validator
func GetValidatorVotingPower(ops HTTPOptions, cfg *config.Config, c client.Client) string {
	bp, err := createBatchPoints(cfg.InfluxDB.Database)
	if err != nil {
		log.Printf("Error: %v", err)
		return ""
	}

	resp, err := HitHTTPTarget(HTTPOptions{
		Endpoint: cfg.LCDEndpoint + "/staking/validators/" + cfg.ValOperatorAddress,
		Method:   http.MethodGet,
	})

	if err != nil {
		log.Printf("Error in validator voting power : %v", err)
		return err.Error()
	}

	if resp.Body == nil {
		return ""
	}

	var validatorResp ValidatorResp
	err = json.Unmarshal(resp.Body, &validatorResp)
	if err != nil {
		log.Printf("Error while unamrshelling ValidatorResp: %v", err)
		return ""
	}

	vp := validatorResp.Result.DelegatorShares
	vp1 := convertValue(vp)
	votingPower1, _ := strconv.ParseFloat(vp1, 64)

	votingPowerFromDb := GetVotingPowerFromDb(cfg, c)
	votingPower2, _ := strconv.ParseFloat(votingPowerFromDb, 64)
	var msg string

	if votingPower1 > votingPower2 {
		msg = fmt.Sprintf("Your validator voting power increased to %s ", vp1)
	} else if votingPower1 < votingPower2 {
		msg = fmt.Sprintf("Your validator voting power decreased to %s ", vp1)
	} else {
		msg = fmt.Sprintf("Your new voting power is %s", vp1)
	}

	_ = writeToInfluxDb(c, bp, "vab_voting_power", map[string]string{}, map[string]interface{}{"power": vp1})
	log.Println("Voting Power \n", vp)

	return msg
}

func convertValue(value string) string {
	bal, _ := strconv.ParseFloat(value, 64)

	a1 := bal / math.Pow(10, 6)
	amount := strconv.FormatFloat(a1, 'f', -1, 64)

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
