package monitor

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"

	"github.com/gorilla/websocket"
)

type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

func init() {
	// WSS check is only valid for RPC service type
	RegisterEndpointCheckWithTypes("wss", WssCheck, []string{"RPC"})
}

func WssCheck(check cfg.Check, endpoint string, service cfg.Service, member cfg.Member) {
	ip4 := member.Service.ServiceIPv4
	ip6 := member.Service.ServiceIPv6

	if ip4 == "" && ip6 == "" {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "No IPv4 or IPv6 configured", nil, false)
		return
	}

	if ip4 != "" {
		runWssSingle(check, endpoint, service, member, ip4, false)
	}

	if ip6 != "" {
		runWssSingle(check, endpoint, service, member, ip6, true)
	}
}

func runWssSingle(check cfg.Check, endpoint string, service cfg.Service, member cfg.Member, ip string, isIPv6 bool) {
	u := max.ParseUrl(endpoint)
	reconstructedURL := fmt.Sprintf("%s%s%s", u.Protocol, u.Domain, u.Directory)

	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			ServerName:         u.Domain,
			InsecureSkipVerify: false,
		},
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, net.JoinHostPort(ip, "443"),
				time.Duration(getIntOption(check.ExtraOptions, "ConnectTimeout", 10))*time.Second)
		},
		HandshakeTimeout: time.Duration(getIntOption(check.ExtraOptions, "ConnectTimeout", 10)) * time.Second,
	}

	c, _, err := dialer.Dial(reconstructedURL, nil)
	if err != nil {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, fmt.Sprintf("Failed to connect on IP=%s => %v", ip, err), nil, isIPv6)
		log.Log(log.Debug, "WSS check failed for %s %s isIPv6=%v success=%v", member.Details.Name, endpoint, isIPv6, false)
		return
	}
	defer c.Close()

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "chain_getBlockHash",
		Params:  []interface{}{"latest"},
		ID:      1,
	}

	if !sendJSONRPCRequest(c, request) {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "Failed to send JSON RPC", nil, isIPv6)
		log.Log(log.Debug, "WSS check failed for %s %s isIPv6=%v success=%v", member.Details.Name, endpoint, isIPv6, false)
		return
	}

	_, _, err = c.ReadMessage()
	if err != nil {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, fmt.Sprintf("Failed to read JSON-RPC response: %v", err), nil, isIPv6)
		log.Log(log.Debug, "WSS check failed for %s %s isIPv6=%v success=%v", member.Details.Name, endpoint, isIPv6, false)
		return
	}

	isFullArchive, err := checkFullArchive(c)
	if err != nil {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, fmt.Sprintf("Full archive check failed: %v", err), nil, isIPv6)
		log.Log(log.Debug, "WSS check failed for %s %s isIPv6=%v success=%v", member.Details.Name, endpoint, isIPv6, false)
		return
	}

	if !isFullArchive {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "Not a full archive node", nil, isIPv6)
		log.Log(log.Debug, "WSS check failed for %s %s isIPv6=%v success=%v", member.Details.Name, endpoint, isIPv6, false)
		return
	}

	isCorrectNetwork, err := checkNetwork(c, service.Configuration.NetworkName, service.Configuration.StateRootHash)
	if err != nil {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, fmt.Sprintf("Network check failed: %v", err), nil, isIPv6)
		log.Log(log.Debug, "WSS check failed for %s %s isIPv6=%v success=%v", member.Details.Name, endpoint, isIPv6, false)
		return
	}

	if !isCorrectNetwork {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "Wrong network", nil, isIPv6)
		log.Log(log.Debug, "WSS check failed for %s %s isIPv6=%v success=%v", member.Details.Name, endpoint, isIPv6, false)
		return
	}

	hasEnoughPeers, isSyncing, err := checkPeers(c)
	if err != nil {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, fmt.Sprintf("Peer check failed: %v", err), nil, isIPv6)
		log.Log(log.Debug, "WSS check failed for %s %s isIPv6=%v success=%v", member.Details.Name, endpoint, isIPv6, false)
		return
	}

	if !hasEnoughPeers || isSyncing {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "Syncing or not enough peers", nil, isIPv6)
		log.Log(log.Debug, "WSS check failed for %s %s isIPv6=%v success=%v", member.Details.Name, endpoint, isIPv6, false)
		return
	}

	log.Log(log.Debug, "WSS check completed for %s %s isIPv6=%v success=%v", member.Details.Name, endpoint, isIPv6, true)
	UpdateEndpointResultLocal(check, member, service, endpoint, true, "",
		map[string]interface{}{
			"Syncing": isSyncing,
			"Peers":   hasEnoughPeers,
			"Network": isCorrectNetwork,
			"Archive": isFullArchive,
		}, isIPv6)
}

// The rest is unchanged
func checkFullArchive(c *websocket.Conn) (bool, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "chain_getBlockHash",
		Params:  []interface{}{0},
		ID:      2,
	}

	if !sendJSONRPCRequest(c, req) {
		return false, fmt.Errorf("failed to send blockHash(0) request")
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		return false, fmt.Errorf("failed to read blockHash(0) response: %v", err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(message, &resp); err != nil {
		return false, err
	}

	result, ok := resp["result"].(string)
	if !ok || result == "" {
		return false, fmt.Errorf("invalid chain_getBlockHash(0) response")
	}

	return true, nil
}

func checkNetwork(c *websocket.Conn, expectedNetwork string, expectedStateRootHash string) (bool, error) {
	// First check the chain name
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "system_chain",
		ID:      3,
	}

	if !sendJSONRPCRequest(c, req) {
		return false, fmt.Errorf("failed to send system_chain request")
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		return false, fmt.Errorf("failed to read system_chain response: %v", err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(message, &resp); err != nil {
		return false, err
	}

	chain, ok := resp["result"].(string)
	if !ok {
		return false, fmt.Errorf("invalid system_chain result")
	}

	if !strings.EqualFold(chain, expectedNetwork) {
		return false, nil
	}

	// If StateRootHash is not configured, skip the check
	if expectedStateRootHash == "" {
		log.Log(log.Debug, "StateRootHash not configured for network %s, skipping check", expectedNetwork)
		return true, nil
	}

	// Check the state root hash of the genesis block (block 0)
	// Get block hash at height 0
	blockHashReq := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "chain_getBlockHash",
		Params:  []interface{}{0},
		ID:      4,
	}

	if !sendJSONRPCRequest(c, blockHashReq) {
		return false, fmt.Errorf("failed to send chain_getBlockHash(0) for genesis block")
	}

	_, blockHashMessage, err := c.ReadMessage()
	if err != nil {
		return false, fmt.Errorf("failed to read chain_getBlockHash(0) response: %v", err)
	}

	var blockHashResp map[string]interface{}
	if err := json.Unmarshal(blockHashMessage, &blockHashResp); err != nil {
		return false, fmt.Errorf("failed to unmarshal genesis block hash response: %v", err)
	}

	genesisBlockHash, ok := blockHashResp["result"].(string)
	if !ok || genesisBlockHash == "" {
		return false, fmt.Errorf("invalid genesis block hash response")
	}

	// Get genesis block header to extract state root
	headerReq := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "chain_getHeader",
		Params:  []interface{}{genesisBlockHash},
		ID:      5,
	}

	if !sendJSONRPCRequest(c, headerReq) {
		return false, fmt.Errorf("failed to send chain_getHeader request for genesis block")
	}

	_, headerMessage, err := c.ReadMessage()
	if err != nil {
		return false, fmt.Errorf("failed to read chain_getHeader response: %v", err)
	}

	var headerResp map[string]interface{}
	if err := json.Unmarshal(headerMessage, &headerResp); err != nil {
		return false, fmt.Errorf("failed to unmarshal genesis header response: %v", err)
	}

	// Extract state root from genesis block header
	header, ok := headerResp["result"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("invalid genesis header response format")
	}

	genesisStateRoot, ok := header["stateRoot"].(string)
	if !ok {
		return false, fmt.Errorf("state root not found in genesis header")
	}

	// Compare genesis state root with expected
	if !strings.EqualFold(genesisStateRoot, expectedStateRootHash) {
		log.Log(log.Warn, "Genesis state root mismatch for %s: expected %s, got %s", expectedNetwork, expectedStateRootHash, genesisStateRoot)
		return false, fmt.Errorf("genesis state root mismatch: expected %s, got %s", expectedStateRootHash, genesisStateRoot)
	}

	log.Log(log.Debug, "Genesis state root hash verified for %s: %s", expectedNetwork, genesisStateRoot)
	return true, nil
}

func checkPeers(c *websocket.Conn) (bool, bool, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "system_health",
		ID:      6,
	}

	if !sendJSONRPCRequest(c, req) {
		return false, false, fmt.Errorf("failed to send system_health request")
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		return false, false, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(message, &resp); err != nil {
		return false, false, err
	}

	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		return false, false, fmt.Errorf("invalid system_health result")
	}

	peersF, ok := result["peers"].(float64)
	if !ok {
		return false, false, fmt.Errorf("invalid peers field")
	}

	syncing, ok := result["isSyncing"].(bool)
	if !ok {
		return false, false, fmt.Errorf("invalid isSyncing field")
	}

	hasEnoughPeers := peersF > 5

	return hasEnoughPeers, syncing, nil
}

func sendJSONRPCRequest(c *websocket.Conn, request JSONRPCRequest) bool {
	data, err := json.Marshal(request)
	if err != nil {
		return false
	}

	if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
		return false
	}

	return true
}
