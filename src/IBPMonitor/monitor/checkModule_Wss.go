package monitor

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	cfg "ibp-geodns/src/common/config"
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
	RegisterEndpointCheck("wss", WssCheck)
}

// WssCheck tries IPv4 if present, and IPv6 if present (similar to ping check).
func WssCheck(check cfg.Check, endpoint string, service cfg.Service, member cfg.Member) {
	ip4 := member.Service.ServiceIPv4
	ip6 := member.Service.ServiceIPv6

	// If no IP is configured, fail immediately.
	if ip4 == "" && ip6 == "" {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "No IPv4 or IPv6 configured", nil, false)
		return
	}

	// Attempt WSS check over IPv4
	if ip4 != "" {
		runWssSingle(check, endpoint, service, member, ip4, false)
	}

	// Attempt WSS check over IPv6
	if ip6 != "" {
		runWssSingle(check, endpoint, service, member, ip6, true)
	}
}

// runWssSingle tries a WSS dial to ip:443, then verifies it's a full archive node, correct network, etc.
// Added isIPv6 parameter to track whether this is an IPv6 check
func runWssSingle(check cfg.Check, endpoint string, service cfg.Service, member cfg.Member, ip string, isIPv6 bool) {
	u := max.ParseUrl(endpoint)
	// Reconstruct the wss://... but substituting the IP for the domain
	// so we dial the correct IP. We'll keep the same path as the original parse.
	reconstructedURL := fmt.Sprintf("%s%s%s", u.Protocol, u.Domain, u.Directory)

	// We'll override the dial target with ip:443
	// But the "ServerName" in TLS config is still the domain
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
		return
	}

	_, _, err = c.ReadMessage()
	if err != nil {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, fmt.Sprintf("Failed to read JSON-RPC response: %v", err), nil, isIPv6)
		return
	}

	isFullArchive, err := checkFullArchive(c)
	if err != nil {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, fmt.Sprintf("Full archive check failed: %v", err), nil, isIPv6)
		return
	}
	if !isFullArchive {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "Not a full archive node", nil, isIPv6)
		return
	}

	isCorrectNetwork, err := checkNetwork(c, service.Configuration.NetworkName)
	if err != nil {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, fmt.Sprintf("Network check failed: %v", err), nil, isIPv6)
		return
	}
	if !isCorrectNetwork {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "Wrong network", nil, isIPv6)
		return
	}

	hasEnoughPeers, isSyncing, err := checkPeers(c)
	if err != nil {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, fmt.Sprintf("Peer check failed: %v", err), nil, isIPv6)
		return
	}
	if !hasEnoughPeers || isSyncing {
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "Syncing or not enough peers", nil, isIPv6)
		return
	}

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

func checkNetwork(c *websocket.Conn, expectedNetwork string) (bool, error) {
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
	return true, nil
}

func checkPeers(c *websocket.Conn) (bool, bool, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "system_health",
		ID:      4,
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
