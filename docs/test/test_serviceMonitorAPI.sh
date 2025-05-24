#!/usr/bin/env bash
#
# Demo script to pull official results from serviceMonitor and print 4 sections:
#   1) Site checks
#   2) Domain checks
#   3) Endpoint checks
#   4) By Member
#
# Usage:
#   ./test_monitor_results.sh [monitor_address] [monitor_port]

MONITOR_ADDR="${1:-127.0.0.1}"
MONITOR_PORT="${2:-6101}"

echo "Pulling official results from serviceMonitor at ${MONITOR_ADDR}:${MONITOR_PORT}..."

JSON="$(curl -s http://${MONITOR_ADDR}:${MONITOR_PORT}/results)"
if [ -z "$JSON" ]; then
  echo "ERROR: No JSON data received."
  exit 1
fi

# Check JSON validity
if ! echo "$JSON" | jq . >/dev/null 2>&1; then
  echo "ERROR: Invalid JSON received:"
  echo "$JSON"
  exit 1
fi

TMPFILE="$(mktemp -t sm_results.XXXXXX)"
echo "$JSON" > "$TMPFILE"

echo "==================================================="
echo "   SERVICE MONITOR RESULTS"
echo "==================================================="

##################################################
# 1) SITES
##################################################
echo
echo "---------- SITES ----------"
echo
# We'll keep it simple: For each item in .SiteResults, print the check name
# then each member with status / error
jq -r '
  (.SiteResults // [])[]
  | .Check.Name as $chk
  | " \($chk) => "
    + (
      ( .Results // [] )
      | map(
          "Member: "
          + (.Member.Details.Name // "???")
          + " Status: \(.Status)"
          + ( if (.ErrorText // "") != "" then " (Err: \(.ErrorText))" else "" end )
        )
      | join(" | ")
    )
' "$TMPFILE"

##################################################
# 2) DOMAINS
##################################################
echo
echo "---------- DOMAINS ----------"
echo
# Group by domain, then list each check's name, members, status
jq -r '
  (.DomainResults // [])
  | group_by(.Domain)[]
  | "Domain: " + (.[0].Domain // "???")
  , (
      map(
        .Check.Name as $checkName
        | "  Check: " + ($checkName // "???")
          + " => "
          + (
            ( .Results // [] )
            | map(
                "Member: " + (.Member.Details.Name // "???")
                + " Status: \(.Status)"
                + ( if (.ErrorText // "") != "" then " (Err: \(.ErrorText))" else "" end )
              )
            | join(" | ")
          )
      )
      | join("\n")
    )
' "$TMPFILE"

##################################################
# 3) ENDPOINTS
##################################################
echo
echo "---------- ENDPOINTS ----------"
echo
# Group by domain, then group by rpcUrl
jq -r '
  (.EndpointResults // [])
  | group_by(.Domain)[]
  | "Domain: " + (.[0].Domain // "???")
  , (
      group_by(.RpcUrl)[]
      | "  RPC: " + (.[0].RpcUrl // "???")
      , (
          map(
            .Check.Name as $chk
            | "    Check: " + ($chk // "???")
              + " => "
              + (
                ( .Results // [] )
                | map(
                    "Member: " + (.Member.Details.Name // "???")
                    + " Status: \(.Status)"
                    + ( if (.ErrorText // "") != "" then " (Err: \(.ErrorText))" else "" end )
                  )
                | join(" | ")
              )
          )
          | join("\n")
        )
    )
' "$TMPFILE"

##################################################
# 4) BY MEMBER
##################################################
echo
echo "---------- BY MEMBER ----------"
echo
# We'll unify site/domain/endpoint data, then group by MemberName
# to list everything for each member.

jq -r '
  def siteItems:
    (.SiteResults // [])[] as $s
    | $s.Check.Name as $chName
    | ($s.Results // [])[]
    | {
        MemberName: (.Member.Details.Name // "???"),
        Type: "site",
        Domain: null,
        RpcUrl: null,
        CheckName: $chName,
        Status: .Status,
        ErrorText: .ErrorText
      };

  def domainItems:
    (.DomainResults // [])[] as $d
    | $d.Check.Name as $chName
    | $d.Domain as $dom
    | ($d.Results // [])[]
    | {
        MemberName: (.Member.Details.Name // "???"),
        Type: "domain",
        Domain: $dom,
        RpcUrl: null,
        CheckName: $chName,
        Status: .Status,
        ErrorText: .ErrorText
      };

  def endpointItems:
    (.EndpointResults // [])[] as $e
    | $e.Check.Name as $chName
    | $e.Domain as $dom
    | $e.RpcUrl as $rpc
    | ($e.Results // [])[]
    | {
        MemberName: (.Member.Details.Name // "???"),
        Type: "endpoint",
        Domain: $dom,
        RpcUrl: $rpc,
        CheckName: $chName,
        Status: .Status,
        ErrorText: .ErrorText
      };

  [
    siteItems,
    domainItems,
    endpointItems
  ]
  | add
  | group_by(.MemberName)[]
  | "Member: " + (.[0].MemberName // "???")
  , (
      group_by(.Type)[]
      | if .[0].Type == "site" then
          "  [Site Checks]" ,
          (
            map(
              "    Check: " + (.CheckName // "???") + " => " + ("\( .Status )")
              + if (.ErrorText // "") != "" then " (Err: \(.ErrorText))" else "" end
            )
            | join("\n")
          )
        elif .[0].Type == "domain" then
          "  [Domain Checks]" ,
          (
            map(
              "    Domain: " + (.Domain // "???") + " | Check: " + (.CheckName // "???")
              + " => " + ("\( .Status )")
              + if (.ErrorText // "") != "" then " (Err: \(.ErrorText))" else "" end
            )
            | join("\n")
          )
        else
          "  [Endpoint Checks]" ,
          (
            map(
              "    Domain: " + (.Domain // "???") + " | RpcUrl: " + (.RpcUrl // "???")
              + " | Check: " + (.CheckName // "???")
              + " => " + ("\( .Status )")
              + if (.ErrorText // "") != "" then " (Err: \(.ErrorText))" else "" end
            )
            | join("\n")
          )
        end
      )
      | join("\n")
  )
' "$TMPFILE"

rm -f "$TMPFILE"
echo
echo "Done."
