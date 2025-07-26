package monitor

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
)

type EthRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type EthRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func init() {
	// ETHRPC check is only valid for ETHRPC service type
	RegisterEndpointCheckWithTypes("ethrpc", EthrpcCheck, []string{"ETHRPC"})
}

func EthrpcCheck(check cfg.Check, endpoint string, service cfg.Service, member cfg.Member) {
	ip4 := member.Service.ServiceIPv4
	ip6 := member.Service.ServiceIPv6

	if ip4 == "" && ip6 == "" {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "No IPv4 or IPv6 configured", nil, false)
		return
	}

	if ip4 != "" {
		runEthrpcSingle(check, endpoint, service, member, ip4, false)
	}
	if ip6 != "" {
		runEthrpcSingle(check, endpoint, service, member, ip6, true)
	}
}

func runEthrpcSingle(check cfg.Check, endpoint string, service cfg.Service, member cfg.Member, ip string, isIPv6 bool) {
	u := max.ParseUrl(endpoint)

	// Convert WSS to HTTPS for ETH RPC
	protocol := u.Protocol
	if strings.HasPrefix(protocol, "wss://") {
		protocol = "https://"
	} else if strings.HasPrefix(protocol, "ws://") {
		protocol = "http://"
	}

	// Reconstruct URL - using domain name, not IP
	reconstructedURL := fmt.Sprintf("%s%s%s", protocol, u.Domain, u.Directory)

	log.Log(log.Debug, "ETHRPC check: endpoint=%s => url=%s (connecting via %s) for %s",
		endpoint, reconstructedURL, ip, member.Details.Name)

	// Create HTTP client with custom transport that redirects to IP
	timeoutSec := getIntOption(check.ExtraOptions, "ConnectTimeout", 10)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			ServerName: u.Domain,
		},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// addr will be "domain:port", we replace with "ip:port"
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			log.Log(log.Debug, "ETHRPC dial: intercepting %s:%s => %s:%s", host, port, ip, port)

			dialer := &net.Dialer{
				Timeout: time.Duration(timeoutSec) * time.Second,
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
		},
	}

	client := &http.Client{
		Timeout:   time.Duration(timeoutSec) * time.Second,
		Transport: transport,
	}

	// Test 1: Check eth_chainId
	chainId, err := ethCall(client, reconstructedURL, "eth_chainId", []interface{}{})
	if err != nil {
		UpdateEndpointResultLocal(check, member, service, endpoint, false,
			fmt.Sprintf("eth_chainId failed: %v", err), nil, isIPv6)
		log.Log(log.Debug, "ETHRPC check failed for %s %s isIPv6=%v - eth_chainId error: %v",
			member.Details.Name, endpoint, isIPv6, err)
		return
	}

	// Test 2: Check eth_blockNumber
	blockNumber, err := ethCall(client, reconstructedURL, "eth_blockNumber", []interface{}{})
	if err != nil {
		UpdateEndpointResultLocal(check, member, service, endpoint, false,
			fmt.Sprintf("eth_blockNumber failed: %v", err), nil, isIPv6)
		log.Log(log.Debug, "ETHRPC check failed for %s %s isIPv6=%v - eth_blockNumber error",
			member.Details.Name, endpoint, isIPv6)
		return
	}

	// Test 3: Check net_version
	netVersion, err := ethCall(client, reconstructedURL, "net_version", []interface{}{})
	if err != nil {
		UpdateEndpointResultLocal(check, member, service, endpoint, false,
			fmt.Sprintf("net_version failed: %v", err), nil, isIPv6)
		log.Log(log.Debug, "ETHRPC check failed for %s %s isIPv6=%v - net_version error",
			member.Details.Name, endpoint, isIPv6)
		return
	}

	// Test 4: Check eth_syncing - must be false
	syncingResult, err := ethCall(client, reconstructedURL, "eth_syncing", []interface{}{})
	if err != nil {
		UpdateEndpointResultLocal(check, member, service, endpoint, false,
			fmt.Sprintf("eth_syncing failed: %v", err), nil, isIPv6)
		log.Log(log.Debug, "ETHRPC check failed for %s %s isIPv6=%v - eth_syncing error",
			member.Details.Name, endpoint, isIPv6)
		return
	}

	// Check if syncing is false
	var syncing bool
	if err := json.Unmarshal(syncingResult, &syncing); err != nil {
		// If it's not a boolean, it might be an object (meaning it's syncing)
		UpdateEndpointResultLocal(check, member, service, endpoint, false,
			"Node is syncing", nil, isIPv6)
		log.Log(log.Debug, "ETHRPC check failed for %s %s isIPv6=%v - Node is syncing",
			member.Details.Name, endpoint, isIPv6)
		return
	}

	if syncing {
		UpdateEndpointResultLocal(check, member, service, endpoint, false,
			"Node is syncing", nil, isIPv6)
		log.Log(log.Debug, "ETHRPC check failed for %s %s isIPv6=%v - Node is syncing",
			member.Details.Name, endpoint, isIPv6)
		return
	}

	// Parse responses
	var chainIdStr string
	json.Unmarshal(chainId, &chainIdStr)

	var blockNumberStr string
	json.Unmarshal(blockNumber, &blockNumberStr)

	var netVersionStr string
	json.Unmarshal(netVersion, &netVersionStr)

	// Verify chain matches expected network
	expectedNetwork := strings.ToLower(service.Configuration.NetworkName)

	// Convert hex chainId to decimal for comparison if needed
	var chainIdDecimal int64
	if strings.HasPrefix(chainIdStr, "0x") {
		chainIdDecimal, _ = strconv.ParseInt(chainIdStr[2:], 16, 64)
	}

	// Check if network matches - compare both net_version and chainId
	networkMatches := false
	if strings.EqualFold(netVersionStr, expectedNetwork) {
		networkMatches = true
	} else if strings.EqualFold(chainIdStr, expectedNetwork) {
		networkMatches = true
	} else if fmt.Sprintf("%d", chainIdDecimal) == expectedNetwork {
		networkMatches = true
	}

	log.Log(log.Debug, "ETHRPC network check for %s: expected=%s, chainId=%s, chainIdDec=%d, netVersion=%s, matches=%v",
		member.Details.Name, expectedNetwork, chainIdStr, chainIdDecimal, netVersionStr, networkMatches)

	if !networkMatches {
		UpdateEndpointResultLocal(check, member, service, endpoint, false,
			fmt.Sprintf("Wrong network: expected %s, got net_version=%s chainId=%s (decimal=%d)",
				expectedNetwork, netVersionStr, chainIdStr, chainIdDecimal), nil, isIPv6)
		log.Log(log.Debug, "ETHRPC check failed for %s %s isIPv6=%v - Wrong network",
			member.Details.Name, endpoint, isIPv6)
		return
	}

	// All checks passed
	dataMap := map[string]interface{}{
		"chainId":     chainIdStr,
		"chainIdDec":  chainIdDecimal,
		"blockNumber": blockNumberStr,
		"netVersion":  netVersionStr,
		"syncing":     false,
		"network":     expectedNetwork,
	}

	UpdateEndpointResultLocal(check, member, service, endpoint, true, "", dataMap, isIPv6)
	log.Log(log.Debug, "ETHRPC check completed for %s %s isIPv6=%v success=%v",
		member.Details.Name, endpoint, isIPv6, true)
}

func ethCall(client *http.Client, url string, method string, params []interface{}) (json.RawMessage, error) {
	request := EthRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var rpcResp EthRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}
