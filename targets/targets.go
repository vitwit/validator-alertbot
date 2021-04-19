package targets

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
	"validator-alertbot/config"

	client "github.com/influxdata/influxdb1-client/v2"
)

type targetRunner struct{}

// NewRunner returns targetRunner
func NewRunner() *targetRunner {
	return &targetRunner{}
}

// Run to run the request
func (m targetRunner) Run(function func(ops HTTPOptions, cfg *config.Config, c client.Client), ops HTTPOptions, cfg *config.Config, c client.Client) {
	function(ops, cfg, c)
}

// InitTargets which returns the targets
//can write all the endpoints here
func InitTargets(cfg *config.Config) *Targets {
	return &Targets{List: []Target{
		{
			ExecutionType: "http",
			Name:          "Send missed blocks lerts",
			HTTPOptions: HTTPOptions{
				Endpoint: cfg.ExternalRPC + "/status",
				Method:   http.MethodGet,
			},
			Func:        MissedBlocks,
			ScraperRate: cfg.Scraper.Rate,
		},
		{
			ExecutionType: "http",
			Name:          "Get Validator status alerting",
			HTTPOptions: HTTPOptions{
				Endpoint: cfg.LCDEndpoint + "/cosmos/staking/v1beta1/validators/" + cfg.ValOperatorAddress,
				Method:   http.MethodGet,
			},
			Func:        ValidatorStatusAlert,
			ScraperRate: cfg.Scraper.ValidatorRate,
		},
		{
			ExecutionType: "http",
			Name:          "Network Latest Block",
			HTTPOptions: HTTPOptions{
				Endpoint: cfg.ExternalRPC + "/status?",
				Method:   http.MethodGet,
			},
			Func:        GetNetworkLatestBlock,
			ScraperRate: cfg.Scraper.Rate,
		},
		{
			ExecutionType: "http",
			Name:          "Net Info URL",
			HTTPOptions: HTTPOptions{
				Endpoint: cfg.ValidatorRPCEndpoint + "/net_info?",
				Method:   http.MethodGet,
			},
			Func:        GetNetInfo,
			ScraperRate: cfg.Scraper.Rate,
		},
		{
			ExecutionType: "http",
			Name:          "Proposals",
			HTTPOptions: HTTPOptions{
				Endpoint: cfg.LCDEndpoint + "/cosmos/gov/v1beta1/proposals",
				Method:   http.MethodGet,
			},
			Func:        GetProposals,
			ScraperRate: cfg.Scraper.Rate,
		},
		{
			ExecutionType: "http",
			Name:          "Get Account Balance",
			HTTPOptions: HTTPOptions{
				Endpoint: cfg.LCDEndpoint + "/cosmos/bank/v1beta1/balances/" + cfg.AccountAddress,
				Method:   http.MethodGet,
			},
			Func:        GetAccountInfo,
			ScraperRate: cfg.Scraper.Rate,
		},
		{
			ExecutionType: "http",
			Name:          "tx alerts",
			HTTPOptions: HTTPOptions{
				Endpoint: cfg.ExternalRPC + "/status",
				Method:   http.MethodGet,
			},
			Func:        TxAlerts,
			ScraperRate: cfg.Scraper.Rate,
		},
		{
			ExecutionType: "http",
			Name:          "Current Rewards Amount",
			HTTPOptions: HTTPOptions{
				Endpoint: cfg.LCDEndpoint + "/cosmos/distribution/v1beta1/delegators/" + cfg.AccountAddress + "/rewards/" + cfg.ValOperatorAddress,
				Method:   http.MethodGet,
			},
			Func:        GetRewradsAndCommission,
			ScraperRate: cfg.Scraper.Rate,
		},
		{
			ExecutionType: "http",
			Name:          "Alert about endpoints status",
			Func:          GetEndpointsStatus,
			ScraperRate:   "600s",
		},
		{
			ExecutionType: "Telegram command",
			Name:          "command based alerts",
			Func:          TelegramAlerting,
			ScraperRate:   "2s",
		},
	}}
}

func addQueryParameters(req *http.Request, queryParams QueryParams) {
	params := url.Values{}
	for key, value := range queryParams {
		params.Add(key, value)
	}
	req.URL.RawQuery = params.Encode()
}

// newHTTPRequest to make a new http request
func newHTTPRequest(ops HTTPOptions) (*http.Request, error) {
	// make new request
	req, err := http.NewRequest(ops.Method, ops.Endpoint, bytes.NewBuffer(ops.Body))
	if err != nil {
		return nil, err
	}

	// Add any query parameters to the URL.
	if len(ops.QueryParams) != 0 {
		addQueryParameters(req, ops.QueryParams)
	}

	return req, nil
}

func makeResponse(res *http.Response) (*PingResp, error) {
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return &PingResp{}, err
	}

	response := &PingResp{
		StatusCode: res.StatusCode,
		Body:       body,
	}
	_ = res.Body.Close()
	return response, nil
}

// HitHTTPTarget to hit the target and get response
func HitHTTPTarget(ops HTTPOptions) (*PingResp, error) {
	req, err := newHTTPRequest(ops)
	if err != nil {
		return nil, err
	}

	httpcli := http.Client{Timeout: time.Duration(5 * time.Second)}
	resp, err := httpcli.Do(req)
	if err != nil {
		return nil, err
	}

	res, err := makeResponse(resp)
	if err != nil {
		return nil, err
	}

	return res, nil
}
