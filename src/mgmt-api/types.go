package api

// ApiRequest is what we parse from the JSON body
type ApiRequest struct {
	Method     string `json:"Method"`
	Action     string `json:"Action"`
	Output     string `json:"Output"`
	MemberName string `json:"MemberName"`
	Year       string `json:"Year"`
	Month      string `json:"Month"`
	StartYear  string `json:"StartYear"`
	StartMonth string `json:"StartMonth"`
	EndYear    string `json:"EndYear"`
	EndMonth   string `json:"EndMonth"`
	AuthKey    string `json:"Authkey"`
}

// ApiResponse is our returned JSON
type ApiResponse struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error"`
}

// CheckResult is used for status checks
type CheckResult struct {
	CheckName string                 `json:"checkName"`
	Status    bool                   `json:"status"`
	ErrorText string                 `json:"errorText"`
	Data      map[string]interface{} `json:"data"`
}
