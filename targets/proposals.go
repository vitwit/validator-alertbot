package targets

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"validator-alertbot/config"

	client "github.com/influxdata/influxdb1-client/v2"
)

// GetValidatorVoted to check validator voted for the proposal or not
func GetValidatorVoted(LCDEndpoint string, proposalID string, accountAddress string) string {
	proposalURL := LCDEndpoint + "/cosmos/gov/v1beta1/proposals" + proposalID + "/votes"
	ops := HTTPOptions{
		Endpoint: proposalURL,
		Method:   http.MethodGet,
	}
	res, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error in get proposal votes: %v", err)
	}

	var voters ProposalVoters
	err = json.Unmarshal(res.Body, &voters)
	if err != nil {
		log.Printf("Error while reading resp of proposal voters : %v ", err)
	}

	validatorVoted := "not voted"
	for _, value := range voters.Result {
		if value.Voter == accountAddress {
			validatorVoted = value.Option
		}
	}
	return validatorVoted
}

// SendVotingPeriodProposalAlerts which send alerts of voting period proposals
func SendVotingPeriodProposalAlerts(LCDEndpoint string, accountAddress string, cfg *config.Config) error {
	proposalURL := LCDEndpoint + "/cosmos/gov/v1beta1/proposals?status=PROPOSAL_STATUS_VOTING_PERIOD"
	ops := HTTPOptions{
		Endpoint: proposalURL,
		Method:   http.MethodGet,
	}
	res, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error in get proposal votes: %v", err)
	}

	var p Proposals
	err = json.Unmarshal(res.Body, &p)
	if err != nil {
		log.Printf("Error while reading resp of proposal voters : %v ", err)
	}

	for _, proposal := range p.Proposals {
		proposalVotesURL := LCDEndpoint + "/cosmos/gov/v1beta1/proposals/" + proposal.ProposalID + "/votes"
		ops := HTTPOptions{
			Endpoint: proposalVotesURL,
			Method:   http.MethodGet,
		}
		res, err := HitHTTPTarget(ops)
		if err != nil {
			log.Printf("Error in get proposal votes: %v", err)
		}

		var voters ProposalVoters
		err = json.Unmarshal(res.Body, &voters)
		if err != nil {
			log.Printf("Error while reading resp of proposal voters : %v ", err)
		}
		var validatorVoted string
		for _, value := range voters.Result {
			if value.Voter == accountAddress {
				validatorVoted = value.Option
			}
		}

		if validatorVoted == "VOTE_OPTION_NO" {
			now := time.Now().UTC()
			votingEndTime, _ := time.Parse(time.RFC3339, proposal.VotingEndTime)
			timeDiff := now.Sub(votingEndTime)
			log.Println("timeDiff...", timeDiff.Hours())

			if timeDiff.Hours() <= 24 {
				_ = SendTelegramAlert(fmt.Sprintf("%s validator has not voted on proposal = %s", cfg.ValidatorName, proposal.ProposalID), cfg)
				_ = SendEmailAlert(fmt.Sprintf("%s validator has not voted on proposal = %s", cfg.ValidatorName, proposal.ProposalID), cfg)

				log.Println("Sent alert of voting period proposals")
			}
		}
	}
	return nil
}

// GetValidatorDeposited to check validator deposited for the proposal or not
func GetValidatorDeposited(LCDEndpoint string, proposalID string, accountAddress string) string {
	proposalURL := LCDEndpoint + "/cosmos/gov/v1beta1/proposals/" + proposalID + "/deposits"
	ops := HTTPOptions{
		Endpoint: proposalURL,
		Method:   http.MethodGet,
	}
	res, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error in get proposal votes: %v", err)
	}

	var depositors Depositors
	err = json.Unmarshal(res.Body, &depositors)
	if err != nil {
		log.Printf("Error while reading resp of proposal voters : %v ", err)
	}

	validateDeposit := "no"
	for _, value := range depositors.Deposits {
		if value.Depositor == accountAddress && len(value.Amount) != 0 {
			validateDeposit = "yes"
		}
	}
	return validateDeposit
}

// GetProposals to get all the proposals and send alerts accordingly
func GetProposals(ops HTTPOptions, cfg *config.Config, c client.Client) {
	bp, err := createBatchPoints(cfg.InfluxDB.Database)
	if err != nil {
		return
	}

	resp, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	var p Proposals
	err = json.Unmarshal(resp.Body, &p)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	for _, proposal := range p.Proposals {
		validatorVoted := GetValidatorVoted(cfg.LCDEndpoint, proposal.ProposalID, cfg.AccountAddress)
		validatorDeposited := GetValidatorDeposited(cfg.LCDEndpoint, proposal.ProposalID, cfg.AccountAddress)
		err = SendVotingPeriodProposalAlerts(cfg.LCDEndpoint, cfg.AccountAddress, cfg)
		if err != nil {
			log.Printf("Error while sending voting period alert: %v", err)
		}

		tag := map[string]string{"id": proposal.ProposalID}
		fields := map[string]interface{}{
			"proposal_id":               proposal.ProposalID,
			"content_type":              proposal.Content.Type,
			"content_value_title":       proposal.Content.Title,
			"content_value_description": proposal.Content.Description,
			"proposal_status":           proposal.Status,
			"final_tally_result":        proposal.FinalTallyResult,
			"submit_time":               GetUserDateFormat(proposal.SubmitTime),
			"deposit_end_time":          GetUserDateFormat(proposal.DepositEndTime),
			"total_deposit":             proposal.TotalDeposit,
			"voting_start_time":         GetUserDateFormat(proposal.VotingStartTime),
			"voting_end_time":           GetUserDateFormat(proposal.VotingEndTime),
			"validator_voted":           validatorVoted,
			"validator_deposited":       validatorDeposited,
		}
		newProposal := false
		proposalStatus := ""
		q := client.NewQuery(fmt.Sprintf("SELECT * FROM vab_proposals WHERE proposal_id = '%s'", proposal.ProposalID), cfg.InfluxDB.Database, "")
		if response, err := c.Query(q); err == nil && response.Error() == nil {
			for _, r := range response.Results {
				if len(r.Series) == 0 {
					newProposal = true
					break
				} else {
					for idx, col := range r.Series[0].Columns {
						if col == "proposal_status" {
							v := r.Series[0].Values[0][idx]
							proposalStatus = fmt.Sprintf("%v", v)
						}
					}
				}
			}

			_, synced := GetNodeSync(cfg, c) // get syncing status
			if newProposal {
				log.Printf("New Proposal Came: %s", proposal.ProposalID)
				_ = writeToInfluxDb(c, bp, "vab_proposals", tag, fields)

				if synced == "1" {
					if proposal.Status == "PROPOSAL_STATUS_REJECTED" || proposal.Status == "PROPOSAL_STATUS_PASSED" {
						_ = SendTelegramAlert(fmt.Sprintf("Proposal "+proposal.Content.Type+" with proposal id = %s has been %s", proposal.ProposalID, proposal.Status), cfg)
						_ = SendEmailAlert(fmt.Sprintf("Proposal "+proposal.Content.Type+" with proposal id = %s has been = %s", proposal.ProposalID, proposal.Status), cfg)
					} else if proposal.Status == "PROPOSAL_STATUS_VOTING_PERIOD" {
						_ = SendTelegramAlert(fmt.Sprintf("Proposal "+proposal.Content.Type+" with proposal id = %s has been moved to %s", proposal.ProposalID, proposal.Status), cfg)
						_ = SendEmailAlert(fmt.Sprintf("Proposal "+proposal.Content.Type+" with proposal id = %s has been moved to %s", proposal.ProposalID, proposal.Status), cfg)
					} else {
						_ = SendTelegramAlert(fmt.Sprintf("A new proposal "+proposal.Content.Type+" has been added to "+proposal.Status+" with proposal id = %s", proposal.ProposalID), cfg)
						_ = SendEmailAlert(fmt.Sprintf("A new proposal "+proposal.Content.Type+" has been added to "+proposal.Status+" with proposal id = %s", proposal.ProposalID), cfg)
					}
				}
			} else {
				q := client.NewQuery(fmt.Sprintf("DELETE FROM vab_proposals WHERE id = '%s'", proposal.ProposalID), cfg.InfluxDB.Database, "")
				if response, err := c.Query(q); err == nil && response.Error() == nil {
					log.Printf("Delete proposal %s from vab_proposals", proposal.ProposalID)
				} else {
					log.Printf("Failed to delete proposal %s from vab_proposals", proposal.ProposalID)
					log.Printf("Reason for proposal deletion failure %v", response)
				}
				log.Printf("Writing the proposal: %s", proposal.ProposalID)
				_ = writeToInfluxDb(c, bp, "vab_proposals", tag, fields)

				if synced == "1" {
					if proposal.Status != proposalStatus {
						if proposal.Status == "PROPOSAL_STATUS_REJECTED" || proposal.Status == "PROPOSAL_STATUS_PASSED" {
							_ = SendTelegramAlert(fmt.Sprintf("Proposal "+proposal.Content.Type+" with proposal id = %s has been %s", proposal.ProposalID, proposal.Status), cfg)
							_ = SendEmailAlert(fmt.Sprintf("Proposal "+proposal.Content.Type+" with proposal id = %s has been = %s", proposal.ProposalID, proposal.Status), cfg)
						} else {
							_ = SendTelegramAlert(fmt.Sprintf("Proposal "+proposal.Content.Type+" with proposal id = %s has been moved to %s", proposal.ProposalID, proposal.Status), cfg)
							_ = SendEmailAlert(fmt.Sprintf("Proposal "+proposal.Content.Type+" with proposal id = %s has been moved to %s", proposal.ProposalID, proposal.Status), cfg)
						}
					}
				}
			}
		}
	}

	// Calling fucntion to delete deposit proposals
	// which are ended
	err = DeleteDepoitEndProposals(cfg, c, p)
	if err != nil {
		log.Printf("Error while deleting proposals")
	}
}

// DeleteDepoitEndProposals to delete proposals from db
//which are not present in lcd resposne
func DeleteDepoitEndProposals(cfg *config.Config, c client.Client, p Proposals) error {
	var ID string
	found := false
	q := client.NewQuery("SELECT * FROM vab_proposals where proposal_status='DepositPeriod'", cfg.InfluxDB.Database, "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		for _, r := range response.Results {
			if len(r.Series) != 0 {
				for idx := range r.Series[0].Values {
					proposalID := r.Series[0].Values[idx][7]
					ID = fmt.Sprintf("%v", proposalID)

					for _, proposal := range p.Proposals {
						if proposal.ProposalID == ID {
							found = true
							break
						} else {
							found = false
						}
					}
					if !found {
						q := client.NewQuery(fmt.Sprintf("DELETE FROM vab_proposals WHERE id = '%s'", ID), cfg.InfluxDB.Database, "")
						if response, err := c.Query(q); err == nil && response.Error() == nil {
							log.Printf("Delete proposal %s from vab_proposals", ID)
							return err
						}
						log.Printf("Failed to delete proposal %s from vab_proposals", ID)
						log.Printf("Reason for proposal deletion failure %v", response)
					}
				}
			}
		}
	}
	return nil
}

// GetUserDateFormat to which returns date in a user friendly
func GetUserDateFormat(timeToConvert string) string {
	time, err := time.Parse(time.RFC3339, timeToConvert)
	if err != nil {
		log.Printf("Error while converting date : %v ", err)
	}
	date := time.Format("Mon Jan _2 15:04:05 2006")
	log.Printf("Converted time into date format : %v ", date)
	return date
}
