#!/usr/bin/env bash
# AWS CLI wrapper for Makefile ECR targets.
#
# Auth precedence:
#   1. AWS_CLI_PROFILE (make var / env) — named profile from ~/.aws/config
#   2. AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY — env keys (AWS_PROFILE ignored)
#   3. AWS_PROFILE — named profile from the shell
#   4. Default credential chain (instance role, SSO cache, etc.)
#
# Account id belongs in Makefile AWS_ACCOUNT_ID, not in AWS_PROFILE.

set -euo pipefail

if [ -n "${AWS_CLI_PROFILE:-}" ]; then
	exec aws --profile "${AWS_CLI_PROFILE}" "$@"
fi

if [ -n "${AWS_ACCESS_KEY_ID:-}" ] && [ -n "${AWS_SECRET_ACCESS_KEY:-}" ]; then
	exec env -u AWS_PROFILE aws "$@"
fi

profile="${AWS_PROFILE:-}"
if [ -n "${profile}" ] && [[ "${profile}" =~ ^[0-9]{12}$ ]]; then
	echo "error: AWS_PROFILE=${profile} looks like an account id, not a ~/.aws/config profile." >&2
	echo "  With env keys set: export AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY (make ignores AWS_PROFILE)." >&2
	echo "  Or: unset AWS_PROFILE | export AWS_PROFILE=my-sso | make build AWS_CLI_PROFILE=my-sso" >&2
	exit 1
fi

exec aws "$@"
