package targets

import (
	"time"
	"validator-alertbot/config"

	client "github.com/influxdata/influxdb1-client/v2"
)

type (
	// QueryParams to map the query params of an url
	QueryParams map[string]string

	// HTTPOptions of a target
	HTTPOptions struct {
		Endpoint    string
		QueryParams QueryParams
		Body        []byte
		Method      string
	}

	// PingResp struct
	PingResp struct {
		StatusCode int
		Body       []byte
	}

	ValidatorResp struct {
		Validator struct {
			OperatorAddress string `json:"operator_address"`
			ConsensusPubkey struct {
				Type string `json:"@type"`
				Key  string `json:"key"`
			} `json:"consensus_pubkey"`
			Jailed            bool      `json:"jailed"`
			Status            string    `json:"status"`
			Tokens            string    `json:"tokens"`
			DelegatorShares   string    `json:"delegator_shares"`
			UnbondingHeight   string    `json:"unbonding_height"`
			UnbondingTime     time.Time `json:"unbonding_time"`
			MinSelfDelegation string    `json:"min_self_delegation"`
		} `json:"validator"`
	}

	// BlockResponse response of a block information
	BlockResponse struct {
		Jsonrpc string `json:"jsonrpc"`
		Result  struct {
			BlockID interface{} `json:"block_id"`
			Block   struct {
				Header interface{} `json:"header"`
				Data   struct {
					Txs []string `json:"txs"`
				} `json:"data"`
				Evidence struct {
					Evidence interface{} `json:"evidence"`
				} `json:"evidence"`
				LastCommit struct {
					Height string `json:"height"`
					// Round      string      `json:"round"`
					BlockID    interface{} `json:"block_id"`
					Signatures []struct {
						// BlockIDFlag      int       `json:"block_id_flag"`
						ValidatorAddress string `json:"validator_address"`
						Timestamp        string `json:"timestamp"`
						Signature        string `json:"signature"`
					} `json:"signatures"`
				} `json:"last_commit"`
			} `json:"block"`
		} `json:"result"`
	}

	// NetworkLatestBlock stores latest block height info
	NetworkLatestBlock struct {
		Result struct {
			SyncInfo struct {
				LatestBlockHeight string `json:"latest_block_height"`
			} `json:"sync_info"`
		} `json:"result"`
	}

	// Target is a structure which holds all the parameters of a target
	//this could be used to write endpoints for each functionality
	Target struct {
		ExecutionType string
		HTTPOptions   HTTPOptions
		Name          string
		Func          func(m HTTPOptions, cfg *config.Config, c client.Client)
		ScraperRate   string
	}

	// Targets list of all the targets
	Targets struct {
		List []Target
	}

	// ValidatorRPCStatus
	ValidatorRPCStatus struct {
		Jsonrpc string `json:"jsonrpc"`
		Result  struct {
			NodeInfo interface{} `json:"node_info"`
			SyncInfo struct {
				LatestBlockHash   string `json:"latest_block_hash"`
				LatestAppHash     string `json:"latest_app_hash"`
				LatestBlockHeight string `json:"latest_block_height"`
				LatestBlockTime   string `json:"latest_block_time"`
				CatchingUp        bool   `json:"catching_up"`
			} `json:"sync_info"`
			ValidatorInfo struct {
				Address string `json:"address"`
				PubKey  struct {
					Type  string `json:"type"`
					Value string `json:"value"`
				} `json:"pub_key"`
				VotingPower string `json:"voting_power"`
			} `json:"validator_info"`
		} `json:"result"`
	}

	// ValidatorsHeight struct which represents the details of validator
	ValidatorsHeight struct {
		Jsonrpc string `json:"jsonrpc"`
		Result  struct {
			BlockHeight string `json:"block_height"`
			Validators  []struct {
				Address string `json:"address"`
				PubKey  struct {
					Type  string `json:"type"`
					Value string `json:"value"`
				} `json:"pub_key"`
				VotingPower      string `json:"voting_power"`
				ProposerPriority string `json:"proposer_priority"`
			} `json:"validators"`
		} `json:"result"`
	}

	// Peer is a structure which holds the info about a peer address
	Peer struct {
		RemoteIP         string      `json:"remote_ip"`
		ConnectionStatus interface{} `json:"connection_status"`
		IsOutbound       bool        `json:"is_outbound"`
		NodeInfo         struct {
			Moniker string `json:"moniker"`
			Network string `json:"network"`
		} `json:"node_info"`
	}

	// NetInfoResult struct
	NetInfoResult struct {
		Listening bool          `json:"listening"`
		Listeners []interface{} `json:"listeners"`
		NumPeers  string        `json:"n_peers"`
		Peers     []Peer        `json:"peers"`
	}

	// NetInfo is a structre which holds the details of address
	NetInfo struct {
		JSONRpc string        `json:"jsonrpc"`
		Result  NetInfoResult `json:"result"`
	}

	// ProposalResultContent struct holds the parameters of a proposal content result
	ProposalResultContent struct {
		Type  string `json:"type"`
		Value struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"value"`
	}

	// ProposalResult struct holds the parameters of proposal result
	ProposalResult struct {
		Content          ProposalResultContent `json:"content"`
		ID               string                `json:"id"`
		ProposalStatus   string                `json:"proposal_status"`
		FinalTallyResult interface{}           `json:"final_tally_result"`
		SubmitTime       string                `json:"submit_time"`
		DepositEndTime   string                `json:"deposit_end_time"`
		TotalDeposit     []interface{}         `json:"total_deposit"`
		VotingStartTime  string                `json:"voting_start_time"`
		VotingEndTime    string                `json:"voting_end_time"`
	}

	// Proposals struct holds result of array of proposals
	Proposals struct {
		Height string           `json:"height"`
		Result []ProposalResult `json:"result"`
	}

	// ProposalVoters struct holds the parameters of proposal voters
	ProposalVoters struct {
		Height string `json:"height"`
		Result []struct {
			ProposalID string `json:"proposal_id"`
			Voter      string `json:"voter"`
			Option     string `json:"option"`
		} `json:"result"`
	}

	// Depositors struct which holds the parameters of depositors
	Depositors struct {
		Height string `json:"height"`
		Result []struct {
			ProposalID string `json:"proposal_id"`
			Depositor  string `json:"depositor"`
			Amount     []struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"amount"`
		} `json:"result"`
	}

	// AccountBalance struct which holds the parameters of an account amount
	AccountBalance struct {
		Balances []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"balances"`
		Pagination interface{} `json:"pagination"`
	}

	// TxHashResp is a struct which holds the response of txhash response
	TxHashResp struct {
		Tx         interface{} `json:"tx"`
		TxResponse struct {
			Height    string `json:"height"`
			Txhash    string `json:"txhash"`
			Codespace string `json:"codespace"`
			Code      int    `json:"code"`
			Data      string `json:"data"`
			RawLog    string `json:"raw_log"`
			Logs      []struct {
				MsgIndex int         `json:"msg_index"`
				Log      string      `json:"log"`
				Events   interface{} `json:"events"`
			} `json:"logs"`
			Info      string `json:"info"`
			GasWanted string `json:"gas_wanted"`
			GasUsed   string `json:"gas_used"`
			Tx        struct {
				Type string `json:"@type"`
				Body struct {
					Messages []struct {
						Type                string `json:"@type"`
						DelegatorAddress    string `json:"delegator_address"`
						ValidatorAddress    string `json:"validator_address"`
						ValidatorSrcAddress string `json:"validator_src_address"`
						ValidatorDstAddress string `json:"validator_dst_address"`
						Amount              struct {
							Denom  string `json:"denom"`
							Amount string `json:"amount"`
						} `json:"amount"`
					} `json:"messages"`
					Memo                        string        `json:"memo"`
					TimeoutHeight               string        `json:"timeout_height"`
					ExtensionOptions            []interface{} `json:"extension_options"`
					NonCriticalExtensionOptions []interface{} `json:"non_critical_extension_options"`
				} `json:"body"`
				AuthInfo   interface{} `json:"auth_info"`
				Signatures []string    `json:"signatures"`
			} `json:"tx"`
			Timestamp time.Time `json:"timestamp"`
		} `json:"tx_response"`
	}

	DistributionRewards struct {
		Height string `json:"height"`
		Result struct {
			OperatorAddress string `json:"operator_address"`
			SelfBondRewards []struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"self_bond_rewards"`
			ValCommission struct {
				Commission []struct {
					Denom  string `json:"denom"`
					Amount string `json:"amoun t"`
				} `json:"cmmission"`
			} `json:"val_commission"`
		} `json:"result"`
	}

	// Rewards is a struct which holds outstanding rewards of a validator
	Rewards struct {
		Rewards struct {
			Rewards []struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"rewards"`
		} `json:"rewards"`
	}

	// Commission is a struct which holds the commission of a validator
	Commission struct {
		Commission struct {
			Commission []struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"commission"`
		} `json:"commission"`
	}
)
