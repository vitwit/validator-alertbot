package targets

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"validator-alertbot/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	client "github.com/influxdata/influxdb1-client/v2"
)

// TelegramAlerting
func TelegramAlerting(ops HTTPOptions, cfg *config.Config, c client.Client) {
	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		log.Fatalf("Please configure telegram bot token :", err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	log.Println("len of update.", len(updates))

	msgToSend := ""

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		if update.Message.Text == "/status" {
			msgToSend = GetStatus(cfg, c)
		} else if update.Message.Text == "/node" {
			msgToSend = NodeStatus(cfg, c)
		} else if update.Message.Text == "/peers" {
			msgToSend = GetPeersCountMsg(cfg, c)
		} else if update.Message.Text == "/balance" {
			msgToSend = GetAccountBal(cfg, c)
		} else if update.Message.Text == "/rewards" {
			msgToSend = GetValRewards(cfg, c)
		} else if update.Message.Text == "/rpc_status" {
			msgToSend = GetEndPointsStatus(cfg)
		} else if update.Message.Text == "/list" {
			msgToSend = GetHelp()
		} else {
			text := strings.Split(update.Message.Text, "")
			if len(text) != 0 {
				if text[0] == "/" {
					msgToSend = "Command not found do /list to know about available commands"
				} else {
					msgToSend = " "
				}
			}
		}

		log.Printf("[%s] %s", update.Message.From.UserName, msgToSend)

		if msgToSend != " " {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgToSend)
			msg.ReplyToMessageID = update.Message.MessageID

			bot.Send(msg)
		}
	}
}

// GetEndPointsStatus retsurns status of the configured endpoints i.e, lcd, val and external rpc.
func GetEndPointsStatus(cfg *config.Config) string {
	ops := HTTPOptions{
		Endpoint: cfg.ExternalRPC + "/status",
		Method:   http.MethodGet,
	}
	var msg string

	_, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error in external rpc: %v", err)
		// _ = SendTelegramAlert(fmt.Sprintf("⛔⛔ Unreachable to EXTERNAL RPC :: %s and the ERROR is : %v", ops.Endpoint, err.Error()), cfg)
		msg = msg + fmt.Sprintf("⛔⛔ Unreachable to EXTERNAL RPC :: %s and the ERROR is : %v\n\n", ops.Endpoint, err.Error())
	} else {
		msg = msg + fmt.Sprintf("EXTERNAL RPC  ✅\n\n")
	}

	ops = HTTPOptions{
		Endpoint: cfg.ValidatorRPCEndpoint + "/net_info?",
		Method:   http.MethodGet,
	}

	_, err = HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error in validator rpc: %v", err)
		msg = msg + fmt.Sprintf("⛔⛔ Unreachable to VALIDATOR RPC :: %s and the ERROR is : %v\n\n", ops.Endpoint, err.Error())
	} else {
		msg = msg + fmt.Sprintf("VALIDATOR RPC  ✅\n\n")
	}

	ops = HTTPOptions{
		Endpoint: cfg.LCDEndpoint + "/node_info",
		Method:   http.MethodGet,
	}

	_, err = HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error in lcd endpoint: %v", err)
		msg = msg + fmt.Sprintf("⛔⛔ Unreachable to LCD ENDPOINT :: %s and the ERROR is : %v\n\n", ops.Endpoint, err.Error())
	} else {
		msg = msg + fmt.Sprintf("LCD ENDPOINT  ✅\n\n")
	}

	return msg
}

// GetHelp returns the msg to show for /help
func GetHelp() string {
	msg := "List of available commands\n /status - returns validator status, voting power, current block height " +
		"and network block height\n /peers - returns number of connected peers\n /node - return status of caught-up\n" +
		"/balance - returns the current balance of your account \n /rewards - returns validator rewards + commission in AKT\n /rpc_status - returns the status of lcd, validator and exteral endpoint\n /list - list out the available commands"

	return msg
}

// GetPeersCountMsg returns the no of peers for /peers
func GetPeersCountMsg(cfg *config.Config, c client.Client) string {
	var msg string

	count := GetPeersCount(cfg, c)
	msg = fmt.Sprintf("No of connected peers %s \n", count)

	return msg
}

// NodeStatus returns the node caught up status /node
func NodeStatus(cfg *config.Config, c client.Client) string {
	var status string

	nodeSync := GetNodeSync(cfg, c)
	status = fmt.Sprintf("Your validator node is %s \n", nodeSync)

	return status
}

// GetStatus returns the status messages for /status
func GetStatus(cfg *config.Config, c client.Client) string {
	var status string

	valStatus := GetValStatusFromDB(cfg, c)
	status = fmt.Sprintf("Your validator is currently  %s \n", valStatus)

	valHeight := GetValidatorBlockHeight(cfg, c)
	status = status + fmt.Sprintf("Validator current block height %s \n", valHeight)

	networkHeight := GetNetworkBlock(cfg, c)
	status = status + fmt.Sprintf("Network current block height %s \n", networkHeight)

	votingPower := GetVotingPowerFromDb(cfg, c)
	status = status + fmt.Sprintf("Voting power of your validator is %s \n", votingPower)

	return status
}

// GetAccountBal returns balance of the corresponding account
func GetAccountBal(cfg *config.Config, c client.Client) string {
	var balanceMsg string

	balance := GetAccountBalFromDb(cfg, c)
	balanceMsg = fmt.Sprintf("Current balance of your account(%s) is %s \n", cfg.AccountAddress, ConvertToAKT(balance, cfg.Denom))

	undelegated, err := GetUndelegated(cfg)
	if err != nil {
		log.Printf("Error while getting undelegated balance : %v", err)
	}

	unbonding := strconv.FormatInt(undelegated, 10)
	balanceMsg = balanceMsg + fmt.Sprintf("Unboding Delegations : %s \n", ConvertToAKT(unbonding, cfg.Denom))

	vp := GetVotingPowerFromDb(cfg, c) + cfg.Denom
	balanceMsg = balanceMsg + fmt.Sprintf("Delegations : %s", vp)
	return balanceMsg
}

// GetValRewards will returns the message of val rewards
func GetValRewards(cfg *config.Config, c client.Client) string {
	var rewardsMsg string
	var ops HTTPOptions

	rewards := GetRewardsFromDB(cfg, c)
	rewardsMsg = fmt.Sprintf("Current rewards and commission of your validator(%s) is :: \nRewards : %s\n", cfg.ValOperatorAddress, rewards)

	commission := GetValCommission(ops, cfg)
	vc := fmt.Sprintf("%f", commission)
	valComm := ConvertToAKT(vc, cfg.Denom)

	rewardsMsg = rewardsMsg + fmt.Sprintf("Commission : %s", valComm)

	return rewardsMsg
}
