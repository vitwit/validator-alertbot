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
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// GetAccountInfo to get account balance information using account address
func GetAccountInfo(ops HTTPOptions, cfg *config.Config, c client.Client) {
	bp, err := createBatchPoints(cfg.InfluxDB.Database)
	if err != nil {
		return
	}

	resp, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	var accResp AccountResp
	err = json.Unmarshal(resp.Body, &accResp)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if len(accResp.Result) > 0 {
		amount := accResp.Result[0].Amount
		denom := accResp.Result[0].Denom
		prevAmount := GetAccountBalFromDb(cfg, c)
		if prevAmount != amount {

			if strings.ToUpper(cfg.BalanceChangeAlerts) == "YES" {
				amount1 := ConvertToAKT(prevAmount)
				amount2 := ConvertToAKT(amount)
				_ = SendTelegramAlert(fmt.Sprintf("Your account balance has changed from  %s to %s", amount1, amount2), cfg)
				_ = SendEmailAlert(fmt.Sprintf("Your account balance has changed from  %s to %s", amount1, amount2), cfg)
			}
		}

		_ = writeToInfluxDb(c, bp, "vcf_account_balance", map[string]string{}, map[string]interface{}{"balance": amount, "denom": denom})
		log.Printf("Address Balance: %s \t and denom : %s", amount, denom)
	}
}

// ConvertToAKT converts balance from uakt to AKT
func ConvertToAKT(balance string) string {
	bal, _ := strconv.ParseFloat(balance, 64)

	a1 := bal / math.Pow(10, 6)
	amount := fmt.Sprintf("%.6f", a1)

	return convertToCommaSeparated(amount) + "AKT"
}

// GetAccountBalFromDb returns account balance from db
func GetAccountBalFromDb(cfg *config.Config, c client.Client) string {
	var balance string
	q := client.NewQuery("SELECT last(balance) FROM vcf_account_balance", cfg.InfluxDB.Database, "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		for _, r := range response.Results {
			if len(r.Series) != 0 {
				for idx, col := range r.Series[0].Columns {
					if col == "last" {
						amount := r.Series[0].Values[0][idx]
						balance = fmt.Sprintf("%v", amount)
						break
					}
				}
			}
		}
	}
	return balance
}

func convertToCommaSeparated(amt string) string {
	a, err := strconv.Atoi(amt)
	if err != nil {
		return amt
	}
	p := message.NewPrinter(language.English)
	return p.Sprintf("%d", a)
}
