#!/usr/bin/env bash
#
# Pull official results from serviceMonitor, print four sections:
#   1) Sites
#   2) Domains
#   3) Endpoints
#   4) By Member (the trickiest part)
#
# Approach for "By Member":
#   - Extract site, domain, endpoint items each to a separate file as arrays
#   - Merge them with `jq -s '.[0] + .[1] + .[2]'`
#   - Then group by .MemberName

MONITOR_ADDR="${1:-127.0.0.1}"
MONITOR_PORT="${2:-6101}"

echo "Pulling official results from serviceMonitor at ${MONITOR_ADDR}:${MONITOR_PORT}..."

RAW_JSON="$(curl -s http://${MONITOR_ADDR}:${MONITOR_PORT}/results)"
if [ -z "$RAW_JSON" ]; then
  echo "ERROR: No JSON data."
  exit 1
fi

# Basic JSON check
if ! echo "$RAW_JSON" | jq . >/dev/null 2>&1; then
  echo "ERROR: invalid JSON"
  echo "$RAW_JSON"
  exit 1
fi

TMP_RAW="$(mktemp -t sm_results.XXXXXX)"
echo "$RAW_JSON" > "$TMP_RAW"

echo "==================================================="
echo " SERVICE MONITOR RESULTS"
echo "==================================================="

########################################
# 1) SITES
########################################
echo
echo "---------- SITES ----------"
echo
jq -r '
  (.SiteResults // [])[]
  | .Check.Name as $check
  | " " + ($check // "?") + " => "
    + (
      (.Results // [])
      | map(
          "Member: " + (.Member.Details.Name // "??")
          + " Status: \(.Status)"
          + (if (.ErrorText // "") != "" then " (Err: \(.ErrorText))" else "" end)
        )
      | join(" | ")
    )
' "$TMP_RAW"

########################################
# 2) DOMAINS
########################################
echo
echo "---------- DOMAINS ----------"
echo
jq -r '
  (.DomainResults // [])
  | group_by(.Domain)[]
  | "Domain: " + (.[0].Domain // "?"),
    (
      map(
        .Check.Name as $ch
        | "  Check: " + ($ch // "?") + " => "
          + (
            (.Results // [])
            | map(
                "Member: " + (.Member.Details.Name // "??")
                + " Status: \(.Status)"
                + (if (.ErrorText // "") != "" then " (Err: \(.ErrorText))" else "" end)
              )
            | join(" | ")
          )
      )
      | join("\n")
    )
' "$TMP_RAW"

########################################
# 3) ENDPOINTS
########################################
echo
echo "---------- ENDPOINTS ----------"
echo
jq -r '
  (.EndpointResults // [])
  | group_by(.Domain)[]
  | "Domain: " + (.[0].Domain // "?"),
    (
      group_by(.RpcUrl)[]
      | "  RPC: " + (.[0].RpcUrl // "?"),
        (
          map(
            .Check.Name as $ch
            | "    Check: " + ($ch // "?") + " => "
              + (
                (.Results // [])
                | map(
                    "Member: " + (.Member.Details.Name // "??")
                    + " Status: \(.Status)"
                    + (if (.ErrorText // "") != "" then " (Err: \(.ErrorText))" else "" end)
                  )
                | join(" | ")
              )
          )
          | join("\n")
        )
    )
' "$TMP_RAW"

########################################
# 4) BY MEMBER
########################################
# We'll produce 3 separate arrays: site.json, domain.json, endpoint.json
# Then unify them into one single array array_unified.json
# Then group that final array by .MemberName

echo
echo "---------- BY MEMBER ----------"
echo

TMP_SITE="$(mktemp -t sm_site.XXXXXX)"
TMP_DOMAIN="$(mktemp -t sm_domain.XXXXXX)"
TMP_ENDPOINT="$(mktemp -t sm_endpt.XXXXXX)"
TMP_UNIFIED="$(mktemp -t sm_unified.XXXXXX)"

# Extract site items
jq '
  (.SiteResults // [])[] as $site
  | $site.Check.Name as $checkName
  | ($site.Results // [])[]
  | {
      "MemberName": (.Member.Details.Name // "??"),
      "Type": "site",
      "Domain": null,
      "RpcUrl": null,
      "CheckName": $checkName,
      "Status": .Status,
      "ErrorText": .ErrorText
    }
' "$TMP_RAW" > "$TMP_SITE"

# Extract domain items
jq '
  (.DomainResults // [])[] as $domRes
  | $domRes.Check.Name as $checkName
  | $domRes.Domain as $dom
  | ($domRes.Results // [])[]
  | {
      "MemberName": (.Member.Details.Name // "??"),
      "Type": "domain",
      "Domain": $dom,
      "RpcUrl": null,
      "CheckName": $checkName,
      "Status": .Status,
      "ErrorText": .ErrorText
    }
' "$TMP_RAW" > "$TMP_DOMAIN"

# Extract endpoint items
jq '
  (.EndpointResults // [])[] as $endRes
  | $endRes.Check.Name as $checkName
  | $endRes.Domain as $dom
  | $endRes.RpcUrl as $rpc
  | ($endRes.Results // [])[]
  | {
      "MemberName": (.Member.Details.Name // "??"),
      "Type": "endpoint",
      "Domain": $dom,
      "RpcUrl": $rpc,
      "CheckName": $checkName,
      "Status": .Status,
      "ErrorText": .ErrorText
    }
' "$TMP_RAW" > "$TMP_ENDPOINT"

# Now unify them as a single JSON array
jq -s '.[0] + .[1] + .[2]' "$TMP_SITE" "$TMP_DOMAIN" "$TMP_ENDPOINT" > "$TMP_UNIFIED"

# Confirm it is valid & is an array
if ! jq type "$TMP_UNIFIED" >/dev/null 2>&1; then
  echo "ERROR: $TMP_UNIFIED not valid JSON"
  cat "$TMP_UNIFIED"
  rm -f "$TMP_SITE" "$TMP_DOMAIN" "$TMP_ENDPOINT" "$TMP_UNIFIED" "$TMP_RAW"
  exit 1
fi

# Group by MemberName
jq -r '
  group_by(.MemberName)[]
  | "Member: " + (.[0].MemberName // "???"),
    (
      group_by(.Type)[]
      | if .[0].Type == "site" then
          "  [Site Checks]",
          (
            map(
              "    Check: " + (.CheckName // "???") + " => \(.Status)"
              + (if (.ErrorText // "") != "" then " (Err: \(.ErrorText))" else "" end)
            )
            | join("\n")
          )
        elif .[0].Type == "domain" then
          "  [Domain Checks]",
          (
            map(
              "    Domain: " + (.Domain // "???")
              + " | Check: " + (.CheckName // "???") + " => \(.Status)"
              + (if (.ErrorText // "") != "" then " (Err: \(.ErrorText))" else "" end)
            )
            | join("\n")
          )
        else
          "  [Endpoint Checks]",
          (
            map(
              "    Domain: " + (.Domain // "???")
              + " | RpcUrl: " + (.RpcUrl // "???")
              + " | Check: " + (.CheckName // "???") + " => \(.Status)"
              + (if (.ErrorText // "") != "" then " (Err: \(.ErrorText))" else "" end)
            )
            | join("\n")
          )
        end
      | .
    )
' "$TMP_UNIFIED"

rm -f "$TMP_SITE" "$TMP_DOMAIN" "$TMP_ENDPOINT" "$TMP_UNIFIED" "$TMP_RAW"
echo
echo "Done."
