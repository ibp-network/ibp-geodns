package monitor

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"common/config"
	g "common/networking/geoip"

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

func WssCheck(check config.Check, endpoint string, service config.Service, member config.Member) {
	u := g.ParseUrl(endpoint)

	reconstructedURL := fmt.Sprintf("%s%s%s", u.Protocol, u.Domain, u.Directory)
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			ServerName:         u.Domain,
			InsecureSkipVerify: false,
		},
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, net.JoinHostPort(member.Service.ServiceIPv4, "443"), time.Duration(getIntOption(check.ExtraOptions, "ConnectTimeout", 10))*time.Second)
		},
		HandshakeTimeout: time.Duration(getIntOption(check.ExtraOptions, "ConnectTimeout", 10)) * time.Second,
	}

	c, _, err := dialer.Dial(reconstructedURL, nil)
	if err != nil {
		go UpdateEndpointResultLocal(check, member, service, u.Domain, endpoint, false, "Failed to connect", nil)
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
		go UpdateEndpointResultLocal(check, member, service, u.Domain, endpoint, false, fmt.Sprintf("Failed to send JSON-RPC request to (Member: %s URL: '%s' Error: %v)", member.Details.Name, endpoint, err), nil)
		return
	}

	_, _, err = c.ReadMessage()
	if err != nil {
		go UpdateEndpointResultLocal(check, member, service, u.Domain, endpoint, false, fmt.Sprintf("Failed to read JSON-RPC response from (Member: %s URL: '%s' Error: %v)", member.Details.Name, endpoint, err), nil)
		return
	}

	isFullArchive, err := checkFullArchive(c)
	if err != nil {
		go UpdateEndpointResultLocal(check, member, service, u.Domain, endpoint, false, fmt.Sprintf("Full archive check failed for (Member: %s URL: '%s' Error: %v)", member.Details.Name, endpoint, err), nil)
		return
	}

	if !isFullArchive {
		go UpdateEndpointResultLocal(check, member, service, u.Domain, endpoint, false, fmt.Sprintf("Endpoint is not a full archive node (Member: %s URL: '%s')", member.Details.Name, endpoint), nil)
		return
	}

	isCorrectNetwork, err := checkNetwork(c, service.Configuration.NetworkName)
	if err != nil {
		go UpdateEndpointResultLocal(check, member, service, u.Domain, endpoint, false, fmt.Sprintf("Network check failed for (Member: %s URL: '%s' Error: %v)", member.Details.Name, endpoint, err), nil)
		return
	}

	if !isCorrectNetwork {
		go UpdateEndpointResultLocal(check, member, service, u.Domain, endpoint, false, fmt.Sprintf("Endpoint is not on the expected network (Member: %s URL: '%s' Error: %v)", member.Details.Name, endpoint, err), nil)
		return
	}

	hasEnoughPeers, isSyncing, err := checkPeers(c)
	if err != nil {
		go UpdateEndpointResultLocal(check, member, service, u.Domain, endpoint, false, fmt.Sprintf("Peers check failed for (Member: %s URL: '%s' Error: %v)", member.Details.Name, endpoint, err), nil)
		return
	}

	if !hasEnoughPeers || isSyncing {
		go UpdateEndpointResultLocal(check, member, service, u.Domain, endpoint, false, fmt.Sprintf("Endpoint has insufficient peers or is syncing (Member: %s URL: '%s')", member.Details.Name, endpoint), nil)
		return
	}

	// If all checks pass
	go UpdateEndpointResultLocal(check, member, service, u.Domain, endpoint, true, "", map[string]interface{}{"Syncing": isSyncing, "Peers": hasEnoughPeers, "Network": isCorrectNetwork, "Archive": isFullArchive})
}

func checkFullArchive(c *websocket.Conn) (bool, error) {
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "chain_getBlockHash",
		Params:  []interface{}{0},
		ID:      2,
	}

	if !sendJSONRPCRequest(c, request) {
		return false, fmt.Errorf("failed to send chain_getBlockHash request")
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		return false, fmt.Errorf("failed to read chain_getBlockHash response: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(message, &response); err != nil {
		return false, fmt.Errorf("failed to unmarshal chain_getBlockHash response: %v", err)
	}

	result, ok := response["result"].(string)
	if !ok || result == "" {
		return false, fmt.Errorf("chain_getBlockHash result is invalid")
	}

	return true, nil
}

func checkNetwork(c *websocket.Conn, expectedNetwork string) (bool, error) {
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "system_chain",
		Params:  []interface{}{},
		ID:      3,
	}

	if !sendJSONRPCRequest(c, request) {
		return false, fmt.Errorf("failed to send system_chain request")
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		return false, fmt.Errorf("failed to read system_chain response: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(message, &response); err != nil {
		return false, fmt.Errorf("failed to unmarshal system_chain response: %v", err)
	}

	chain, ok := response["result"].(string)
	if !ok || chain == "" {
		return false, fmt.Errorf("system_chain result is invalid")
	}

	if !strings.EqualFold(chain, expectedNetwork) {
		return false, fmt.Errorf("node is on '%s' network instead of expected '%s'", chain, expectedNetwork)
	}

	return true, nil
}

func checkPeers(c *websocket.Conn) (bool, bool, error) {
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "system_health",
		Params:  []interface{}{},
		ID:      4,
	}

	if !sendJSONRPCRequest(c, request) {
		return false, false, fmt.Errorf("failed to send system_health request")
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		return false, false, fmt.Errorf("failed to read system_health response: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(message, &response); err != nil {
		return false, false, fmt.Errorf("failed to unmarshal system_health response: %v", err)
	}

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		return false, false, fmt.Errorf("system_health result is invalid")
	}

	peers, ok := result["peers"].(float64)
	if !ok {
		return false, false, fmt.Errorf("peers count is invalid")
	}

	isSyncing, ok := result["isSyncing"].(bool)
	if !ok {
		return false, false, fmt.Errorf("isSyncing status is invalid")
	}

	hasEnoughPeers := peers > 5
	return hasEnoughPeers, isSyncing, nil
}

func sendJSONRPCRequest(c *websocket.Conn, request JSONRPCRequest) bool {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return false
	}

	if err := c.WriteMessage(websocket.TextMessage, requestBytes); err != nil {
		return false
	}

	return true
}
