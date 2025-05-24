#!/usr/bin/env bash
#
# testapi_by_member.sh
# Pull official results from the serviceMonitor, then group and display them by Member.

MON_ADDR="${1:-127.0.0.1}"
MON_PORT="${2:-6101}"

echo "Pulling official results from serviceMonitor at ${MON_ADDR}:${MON_PORT}..."

RAW_JSON="$(curl -s http://${MON_ADDR}:${MON_PORT}/results)"
if [ -z "$RAW_JSON" ]; then
  echo "ERROR: No JSON data from serviceMonitor."
  exit 1
fi

# Validate JSON quickly
if ! echo "$RAW_JSON" | jq . >/dev/null 2>&1; then
  echo "ERROR: Invalid JSON in response."
  echo "$RAW_JSON"
  exit 1
fi

# If there's a top-level "Result", unwrap it
IS_WRAPPED="$(echo "$RAW_JSON" | jq 'has("Result")')"
if [ "$IS_WRAPPED" = "true" ]; then
  PARSED_JSON="$(echo "$RAW_JSON" | jq '.Result')"
else
  PARSED_JSON="$RAW_JSON"
fi

TMP_PARSED="$(mktemp -t sm_parsed.XXXXXX)"
echo "$PARSED_JSON" > "$TMP_PARSED"

TMP_SITE="$(mktemp -t sm_site.XXXXXX)"
TMP_DOMAIN="$(mktemp -t sm_domain.XXXXXX)"
TMP_ENDPOINT="$(mktemp -t sm_endpt.XXXXXX)"
TMP_MERGED="$(mktemp -t sm_merged.XXXXXX)"

##############################################################################
# Flatten site results into an array
##############################################################################
jq '
  [
    (.SiteResults // [])[]
    | select(type=="object")
    | select(has("Check") and (.Check|type=="object"))
    | select(has("Results"))
    | . as $obj
    | $obj.Check.Name as $checkName
    | ($obj.Results // [])[]
    | select(type=="object")
    | select(has("Member"))
    | select(.Member|type=="object")
    | select(.Member|has("Details"))
    | select(.Member.Details|type=="object")
    | {
        MemberName: (.Member.Details.Name // "???"),
        Type: "site",
        Domain: null,
        RpcUrl: null,
        CheckName: $checkName,
        Status: .Status,
        ErrorText: (.ErrorText // "")
      }
  ]
' "$TMP_PARSED" > "$TMP_SITE"

##############################################################################
# Flatten domain results into an array
##############################################################################
jq '
  [
    (.DomainResults // [])[]
    | select(type=="object")
    | select(has("Check") and (.Check|type=="object"))
    | select(has("Domain") and has("Results"))
    | . as $obj
    | $obj.Check.Name as $checkName
    | $obj.Domain as $domainName
    | ($obj.Results // [])[]
    | select(type=="object")
    | select(has("Member"))
    | select(.Member|type=="object")
    | select(.Member|has("Details"))
    | select(.Member.Details|type=="object")
    | {
        MemberName: (.Member.Details.Name // "???"),
        Type: "domain",
        Domain: $domainName,
        RpcUrl: null,
        CheckName: $checkName,
        Status: .Status,
        ErrorText: (.ErrorText // "")
      }
  ]
' "$TMP_PARSED" > "$TMP_DOMAIN"

##############################################################################
# Flatten endpoint results into an array
##############################################################################
jq '
  [
    (.EndpointResults // [])[]
    | select(type=="object")
    | select(has("Check") and (.Check|type=="object"))
    | select(has("Domain") and has("RpcUrl") and has("Results"))
    | . as $obj
    | $obj.Check.Name as $checkName
    | $obj.Domain as $domainName
    | $obj.RpcUrl as $rpcUrl
    | ($obj.Results // [])[]
    | select(type=="object")
    | select(has("Member"))
    | select(.Member|type=="object")
    | select(.Member|has("Details"))
    | select(.Member.Details|type=="object")
    | {
        MemberName: (.Member.Details.Name // "???"),
        Type: "endpoint",
        Domain: $domainName,
        RpcUrl: $rpcUrl,
        CheckName: $checkName,
        Status: .Status,
        ErrorText: (.ErrorText // "")
      }
  ]
' "$TMP_PARSED" > "$TMP_ENDPOINT"

# Merge the three arrays into one big array
jq -s '[ .[0][] , .[1][] , .[2][] ]' "$TMP_SITE" "$TMP_DOMAIN" "$TMP_ENDPOINT" > "$TMP_MERGED"

echo "==================================================="
echo " SERVICE MONITOR RESULTS (Grouped by Member)"
echo "==================================================="

jq -r '
  # We have a single array of objects. Filter for valid objects that have .MemberName.
  map(select(type=="object" and has("MemberName") and (.MemberName|type=="string")))
  | group_by(.MemberName)[] as $group
  | "Member: " + ($group[0].MemberName // "???")
  + (
      # Site checks
      (
        $group
        | map(select(.Type=="site"))
        | if length>0 then
            "\n  [Site Checks]\n"
            +
            (
              map(
                "    " + (.CheckName // "?") + " => " + (if .Status then "true" else "false" end)
                + if (.ErrorText|length)>0 then " (Err: " + .ErrorText + ")" else "" end
              )
              | join("\n")
            )
          else
            ""
          end
      )
      +
      # Domain checks
      (
        $group
        | map(select(.Type=="domain"))
        | if length>0 then
            "\n  [Domain Checks]\n"
            +
            (
              map(
                "    Domain: " + (.Domain // "?")
                + " | Check: " + (.CheckName // "?")
                + " => " + (if .Status then "true" else "false" end)
                + if (.ErrorText|length)>0 then " (Err: " + .ErrorText + ")" else "" end
              )
              | join("\n")
            )
          else
            ""
          end
      )
      +
      # Endpoint checks
      (
        $group
        | map(select(.Type=="endpoint"))
        | if length>0 then
            "\n  [Endpoint Checks]\n"
            +
            (
              map(
                "    Domain: " + (.Domain // "?")
                + (if (.RpcUrl|length)>0 then " | RPC: " + .RpcUrl else "" end)
                + " | Check: " + (.CheckName // "?")
                + " => " + (if .Status then "true" else "false" end)
                + if (.ErrorText|length)>0 then " (Err: " + .ErrorText + ")" else "" end
              )
              | join("\n")
            )
          else
            ""
          end
      )
    )
  + "\n"
' "$TMP_MERGED"

echo
echo "Done."

rm -f "$TMP_SITE" "$TMP_DOMAIN" "$TMP_ENDPOINT" "$TMP_MERGED" "$TMP_PARSED"