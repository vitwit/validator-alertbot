// adding cons struct
package config

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/go-playground/validator.v9"
)

type (
	// Telegram bot config details
	TelegramBotConfig struct {
		BotToken string `mapstructure:"tg_bot_token"`
		ChatID   int64  `mapstructure:"tg_chat_id"`
	}

	// EmailConfig
	EmailConfig struct {
		SendGridAPIToken    string `mapstructure:"sendgrid_token"`
		ReceiverMailAddress string `mapstructure:"email_address"`
	}

	//InfluxDB details
	InfluxDB struct {
		Port     string `mapstructure:"port"`
		Database string `mapstructure:"database"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
	}

	//Scraper time interval
	Scraper struct {
		Rate                string `mapstructure:"rate"`
		Port                string `mapstructure:"port"`
		ValidatorRate       string `mapstructure:"validator_rate"`
		MissedBlockInterval string `mapstructure:"missed_block_interval"`
		IndexOffSetInterval string `mapstructure:"index_offset_interval"`
	}

	// BlockDiffAlert defines about block diff alert
	BlockDiffAlert struct {
		EnableAlert        string `mapstructure:"enable_alert"`
		BlockDiffThreshold int64  `mapstructure:"block_diff_threshold"`
	}

	// VotingPowerAlert defines about voting power alert
	VotingPowerAlert struct {
		EnableAlert          string `mapstructure:"enable_alert"`
		VotingPowerThreshold int64  `mapstructure:"voting_power_threshold"`
	}

	// PeersAlert defines about peer alerts
	PeersAlert struct {
		EnableAlert       string `mapstructure:"enable_alert"`
		NumPeersThreshold int64  `mapstructure:"num_peers_threshold"`
	}

	// MissedBlocksAlert is about sending alerts of missed blocks based on configuration
	MissedBlocksAlert struct {
		EnableAlert           string `mapstructure:"enable_alert"`
		MissedBlocksThreshold int64  `mapstructure:"missed_blocks_threshold"`
		IndexOffSetThreshold  int64  `mapstructure:"index_offset_threshold"`
	}

	DelegationAlerts struct {
		DelegationAmountThreshold float64 `mapstructure:"delegation_amount_threshold"`
		AccBalanceChangeThreshold float64 `mapstructure:"acc_balance_change_threshold"`
	}

	EnableAlerts struct {
		EnableTelegramAlerts string `mapstructure:"enable_telegram_alerts"`
		EnableEmailAlerts    string `mapstructure:"enable_email_alerts"`
	}

	NodeSyncAlerts struct {
		EnableAlerts string `mapstructure:"enable_alerts"`
	}

	// Config defines all the app configurations
	Config struct {
		ValidatorRPCEndpoint string            `mapstructure:"validator_rpc_endpoint"`
		ValOperatorAddress   string            `mapstructure:"val_operator_addr"`
		ValidatorConsAddress string            `mapstructure:"validator_cons_addr"`
		LCDEndpoint          string            `mapstructure:"lcd_endpoint"`
		Denom                string            `mapstructure:"denom"`
		EnableAlerts         EnableAlerts      `mapstructure:"enable_alerts"`
		Telegram             TelegramBotConfig `mapstructure:"telegram"`
		SendGrid             EmailConfig       `mapstructure:"sendgrid"`
		NodeSyncAlerts       NodeSyncAlerts    `mapstructure:"node_sync_alerts"`
		ExternalRPC          string            `mapstructure:"external_rpc"`
		AlertTime1           string            `mapstructure:"alert_time1"`
		AlertTime2           string            `mapstructure:"alert_time2"`
		ValidatorName        string            `mapstructure:"validator_name"`
		InfluxDB             InfluxDB          `mapstructure:"influxdb"`
		Scraper              Scraper           `mapstructure:"scraper"`
		BlockDiffAlert       BlockDiffAlert    `mapstructure:"block_diff_alert"`
		VotingPowerAlert     VotingPowerAlert  `mapstructure:"voting_power_alert"`
		PeersAlert           PeersAlert        `mapstructure:"Peers_alert"`
		AccountAddress       string            `mapstructure:"account_addr"`
		BalanceChangeAlerts  string            `mapstructure:"balance_change_alert"`
		BalanceDenom         string            `mapstructure:"balance_denom"`
		MissedBlocksAlert    MissedBlocksAlert `mapstructure:"missed_blocks_alert"`
		DelegationAlerts     DelegationAlerts  `mapstructure:"delegation_alerts"`
	}
)

// ReadConfigFromFile to read config details using viper
func ReadConfigFromFile() (*Config, error) {
	v := viper.New()
	v.AddConfigPath(".")
	v.AddConfigPath("./config/")
	v.SetConfigName("config")
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("error while reading config.toml: %v", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("error unmarshaling config.toml to application config: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("error occurred in config validation: %v", err)
	}

	return &cfg, nil
}

// Validate config struct
func (c *Config) Validate(e ...string) error {
	v := validator.New()
	if len(e) == 0 {
		return v.Struct(c)
	}
	return v.StructExcept(c, e...)
}
