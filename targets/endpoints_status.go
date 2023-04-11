package targets

import (
	"fmt"
	"log"
	"net/http"
	"validator-alertbot/config"

	client "github.com/influxdata/influxdb1-client/v2"
)

// GetEndpointsStatus to get alert about endpoints status
func GetEndpointsStatus(ops HTTPOptions, cfg *config.Config, c client.Client) {
	var msg string

	ops = HTTPOptions{
		Endpoint: cfg.LCDEndpoint + "/cosmos/slashing/v1beta1/signing_infos/" + cfg.ValidatorConsAddress,
		//Endpoint: cfg.ExternalRPC + "/status",
		Method: http.MethodGet,
	}
	_, err := HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error in external rpc: %v", err)
		msg = msg + fmt.Sprintf("⛔⛔ Unreachable to EXTERNAL RPC :: %s and the ERROR is : %v\n\n", ops.Endpoint, err.Error())
	}

	ops = HTTPOptions{
		Endpoint: cfg.ValidatorRPCEndpoint + "/net_info?",
		Method:   http.MethodGet,
	}

	_, err = HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error in validator rpc: %v", err)
		msg = msg + fmt.Sprintf("⛔⛔ Unreachable to VALIDATOR RPC :: %s and the ERROR is : %v\n\n", ops.Endpoint, err.Error())
	}

	ops = HTTPOptions{
		Endpoint: cfg.LCDEndpoint + "/node_info",
		Method:   http.MethodGet,
	}

	_, err = HitHTTPTarget(ops)
	if err != nil {
		log.Printf("Error in lcd endpoint: %v", err)
		msg = msg + fmt.Sprintf("⛔⛔ Unreachable to LCD ENDPOINT :: %s and the ERROR is : %v\n\n", ops.Endpoint, err.Error())
	}

	if msg != "" {
		_ = SendTelegramAlert(msg, cfg)
	}
}
