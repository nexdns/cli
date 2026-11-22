#!/usr/bin/env sh
# shellcheck disable=SC2034
dns_nexdns_info='NexDNS
Site: nexdns.tech
Docs: github.com/acmesh-official/acme.sh/wiki/dnsapi2#dns_nexdns
Options:
 NEXDNS_Token API token. Can be created at https://nexdns.tech/settings/api-keys
 NEXDNS_Api API base url. Default "https://api.nexdns.tech/v1". Optional.
Issues: github.com/acmesh-official/acme.sh/issues/7179
Author: NexDNS <https://github.com/nexdns>
'

NEXDNS_Api_Default="https://api.nexdns.tech/v1"

########  Public functions #####################

#Usage: dns_nexdns_add   _acme-challenge.www.example.com   "XKrxpRBosdIKFzxW_CT3KLZNf6q0HG9i01zxXp5CPBs"
dns_nexdns_add() {
  fulldomain=$1
  txtvalue=$2

  if ! _nexdns_init; then
    return 1
  fi

  _saveaccountconf_mutable NEXDNS_Token "$NEXDNS_Token"
  if [ "$NEXDNS_Api" != "$NEXDNS_Api_Default" ]; then
    _saveaccountconf_mutable NEXDNS_Api "$NEXDNS_Api"
  else
    _clearaccountconf_mutable NEXDNS_Api
  fi

  _debug "First detect the root zone"
  if ! _get_root "$fulldomain"; then
    _err "Cannot find the zone of $fulldomain in this NexDNS account."
    return 1
  fi
  _debug _sub_domain "$_sub_domain"
  _debug _domain "$_domain"
  _debug _domain_id "$_domain_id"

  _info "Adding the TXT record for $fulldomain"
  if ! _nexdns_rest POST "zones/$_domain_id/records" "{\"name\":\"$_sub_domain\",\"type\":\"TXT\",\"content\":\"$txtvalue\",\"ttl\":120}"; then
    return 1
  fi

  _info "The TXT record has been added."
  return 0
}

#Usage: dns_nexdns_rm   _acme-challenge.www.example.com   "XKrxpRBosdIKFzxW_CT3KLZNf6q0HG9i01zxXp5CPBs"
dns_nexdns_rm() {
  fulldomain=$1
  txtvalue=$2

  if ! _nexdns_init; then
    return 1
  fi

  _debug "First detect the root zone"
  if ! _get_root "$fulldomain"; then
    _err "Cannot find the zone of $fulldomain in this NexDNS account."
    return 1
  fi
  _debug _sub_domain "$_sub_domain"
  _debug _domain "$_domain"
  _debug _domain_id "$_domain_id"

  _info "Removing the TXT record for $fulldomain"
  if ! _nexdns_rest GET "zones/$_domain_id/records?type=TXT&name=$_sub_domain"; then
    return 1
  fi

  #example.com and *.example.com put two challenge records at one name, so match the value.
  _record_id="$(echo "$response" | tr '{' "\n" | grep -- "$txtvalue" | _egrep_o '"id":"[^"]*"' | _head_n 1 | cut -d '"' -f 4)"
  _debug _record_id "$_record_id"

  if [ -z "$_record_id" ]; then
    _info "The TXT record is already gone, nothing to remove."
    return 0
  fi

  if ! _nexdns_rest DELETE "zones/$_domain_id/records/$_record_id"; then
    return 1
  fi

  _info "The TXT record has been removed."
  return 0
}

####################  Private functions below ##################################

#Reads the token and the api url, and applies the default url.
_nexdns_init() {
  NEXDNS_Token="${NEXDNS_Token:-$(_readaccountconf_mutable NEXDNS_Token)}"
  NEXDNS_Api="${NEXDNS_Api:-$(_readaccountconf_mutable NEXDNS_Api)}"

  if [ -z "$NEXDNS_Token" ]; then
    _err "You have not set NEXDNS_Token yet."
    _err "Create one at https://nexdns.tech/settings/api-keys, on a plan that includes API access, then:"
    _err "export NEXDNS_Token=\"your-api-token\""
    return 1
  fi

  if [ -z "$NEXDNS_Api" ]; then
    NEXDNS_Api="$NEXDNS_Api_Default"
  fi
  #A trailing slash would make every request path begin with a double slash.
  NEXDNS_Api="$(echo "$NEXDNS_Api" | sed 's|/*$||')"
  _debug NEXDNS_Api "$NEXDNS_Api"

  return 0
}

#_acme-challenge.www.example.com
#returns
# _sub_domain=_acme-challenge.www
# _domain=example.com
# _domain_id=Zm9vYmFy
_get_root() {
  domain=$1
  i=1
  p=1

  while true; do
    h=$(printf "%s" "$domain" | cut -d . -f "$i"-100)
    _debug h "$h"
    if [ -z "$h" ]; then
      #not valid
      return 1
    fi

    if ! _nexdns_rest GET "zones?search=$h&per_page=100"; then
      return 1
    fi

    #search matches substrings; take the zone whose name is exactly h.
    _domain_id="$(echo "$response" | tr '{' "\n" | grep "\"name\":\"$h\"" | _egrep_o '"id":"[^"]*"' | _head_n 1 | cut -d '"' -f 4)"
    if [ "$_domain_id" ]; then
      _sub_domain=$(printf "%s" "$domain" | cut -d . -f 1-"$p")
      _domain=$h
      return 0
    fi

    p=$i
    i=$(_math "$i" + 1)
  done
}

#Usage: _nexdns_rest  GET|POST|DELETE  path  [body]  [attempt]
_nexdns_rest() {
  m=$1
  ep=$2
  data=$3
  attempt=${4:-1}
  _debug "$ep"

  export _H1="Authorization: Bearer $NEXDNS_Token"
  export _H2="Content-Type: application/json"
  export _H3="Accept: application/json"

  if [ "$m" = "GET" ]; then
    response="$(_get "$NEXDNS_Api/$ep")"
  else
    _debug2 data "$data"
    response="$(_post "$data" "$NEXDNS_Api/$ep" "" "$m" "application/json")"
  fi

  if [ "$?" != "0" ]; then
    _err "error $ep"
    return 1
  fi

  #Rate limited: wait and retry. Retry-After is a floor; the backoff grows per attempt.
  if [ "$(grep "^HTTP" "$HTTP_HEADER" 2>/dev/null | _tail_n 1 | cut -d " " -f 2 | tr -d "\r\n")" = "429" ]; then
    if [ "$attempt" -ge 4 ]; then
      _err "$m $ep failed: rate limited, and the wait budget is spent"
      return 1
    fi

    _retry_after="$(grep -i "^Retry-After" "$HTTP_HEADER" 2>/dev/null | _tail_n 1 | cut -d : -f 2 | tr -d " \r\n")"
    _backoff="$(_math "$attempt" \* 15)"
    #Retry-After may be an http date; anything but plain seconds uses the backoff.
    case "$_retry_after" in
    "" | *[!0-9]*) _retry_after="$_backoff" ;;
    *)
      if [ "$_retry_after" -lt "$_backoff" ]; then
        _retry_after="$_backoff"
      fi
      ;;
    esac

    #Do not hold the hook for a long window; the next cron run retries.
    if [ "$_retry_after" -gt 120 ]; then
      _err "$m $ep failed: rate limited for ${_retry_after}s, longer than this hook will wait"
      return 1
    fi

    _info "Rate limited by the NexDNS API; retrying in $_retry_after seconds."
    _sleep "$_retry_after"

    _nexdns_rest "$m" "$ep" "$data" "$(_math "$attempt" + 1)"
    return $?
  fi

  #Strip whitespace so the matches here and in the callers work.
  response="$(echo "$response" | _normalizeJson)"
  _debug2 response "$response"

  #Success is decided by the status code: a delete returns 204 with no body, and
  #a record value may itself contain "error".
  _code="$(grep "^HTTP" "$HTTP_HEADER" 2>/dev/null | _tail_n 1 | cut -d " " -f 2 | tr -d "\r\n")"
  _debug2 _code "$_code"
  case "$_code" in
  "" | 2*)
    return 0
    ;;
  esac

  #Error body: {"error":{"code":..,"message":..}}
  _message="$(echo "$response" | _egrep_o '"message":"[^"]*"' | _head_n 1 | cut -d '"' -f 4)"
  if [ -z "$_message" ]; then
    _message="status $_code"
  fi
  _err "$m $ep failed: $_message"

  return 1
}
