#!/usr/bin/env bash
# Deploy pure-Go Lambda zip + public Function URL (no Docker/ECR).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REGION="${AWS_REGION:-$(aws configure get region 2>/dev/null || true)}"
REGION="${REGION:-us-east-1}"
FUNCTION="${FUNCTION:-tela-gateway}"
ARCH="${LAMBDA_ARCH:-arm64}"
ROLE_NAME="${FUNCTION}-role"
ZIP="${ROOT}/dist/function.zip"

export AWS_REGION="${REGION}"
export AWS_DEFAULT_REGION="${REGION}"

ACCOUNT="$(aws sts get-caller-identity --query Account --output text)"
ROLE_ARN="arn:aws:iam::${ACCOUNT}:role/${ROLE_NAME}"

echo "Account:  ${ACCOUNT}"
echo "Region:   ${REGION}"
echo "Function: ${FUNCTION}"
echo "Arch:     ${ARCH}"

"${ROOT}/scripts/build-lambda-zip.sh"

ZIP_BYTES="$(wc -c < "${ZIP}" | tr -d ' ')"
if [[ "${ZIP_BYTES}" -gt 52428800 ]]; then
	echo "Zip is >50MB (${ZIP_BYTES} bytes). Upload via S3 is required — not implemented in this script." >&2
	exit 1
fi

if ! aws iam get-role --role-name "${ROLE_NAME}" >/dev/null 2>&1; then
	echo "Creating IAM role ${ROLE_NAME}..."
	aws iam create-role --role-name "${ROLE_NAME}" \
		--assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}' \
		>/dev/null
	aws iam attach-role-policy --role-name "${ROLE_NAME}" \
		--policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
	echo "Waiting for IAM propagation..."
	sleep 12
fi

# Comma-separated daemon list breaks the inline CLI syntax, so use a JSON file.
ENV_FILE="$(mktemp)"
cat > "${ENV_FILE}" <<EOF
{"Variables":{"DERO_DAEMON_URLS":"node.derofoundation.org:11012,dero.rabidmining.com:10102,dero-node.net:10102,community-pools.mysrv.cloud:10102","TELA_DATA_DIR":"/tmp/tela-gateway","TELA_HOST_SUFFIX":".tela.cypher-punks.com","TELA_ALIASES":"","ALIASES_S3_URI":"s3://cypher-punks-tela-aliases/aliases.json"}}
EOF
ENV_VARS="file://${ENV_FILE}"

if aws lambda get-function --function-name "${FUNCTION}" --region "${REGION}" >/dev/null 2>&1; then
	echo "Updating function code..."
	aws lambda update-function-code \
		--function-name "${FUNCTION}" \
		--zip-file "fileb://${ZIP}" \
		--region "${REGION}" >/dev/null
	aws lambda wait function-updated --function-name "${FUNCTION}" --region "${REGION}"
	aws lambda update-function-configuration \
		--function-name "${FUNCTION}" \
		--timeout 120 \
		--memory-size 1024 \
		--ephemeral-storage Size=1024 \
		--environment "${ENV_VARS}" \
		--region "${REGION}" >/dev/null
	aws lambda wait function-updated --function-name "${FUNCTION}" --region "${REGION}"
else
	echo "Creating function..."
	aws lambda create-function \
		--function-name "${FUNCTION}" \
		--runtime provided.al2023 \
		--handler bootstrap \
		--architectures "${ARCH}" \
		--role "${ROLE_ARN}" \
		--zip-file "fileb://${ZIP}" \
		--timeout 120 \
		--memory-size 1024 \
		--ephemeral-storage Size=1024 \
		--environment "${ENV_VARS}" \
		--region "${REGION}" >/dev/null
	aws lambda wait function-active-v2 --function-name "${FUNCTION}" --region "${REGION}"
fi

if ! aws lambda get-function-url-config --function-name "${FUNCTION}" --region "${REGION}" >/dev/null 2>&1; then
	echo "Creating public Function URL..."
	aws lambda create-function-url-config \
		--function-name "${FUNCTION}" \
		--auth-type NONE \
		--region "${REGION}" >/dev/null
fi

# Resource-based policy for public Function URL + direct invoke
aws lambda add-permission \
	--function-name "${FUNCTION}" \
	--statement-id FunctionURLAllowPublicAccess \
	--action lambda:InvokeFunctionUrl \
	--principal "*" \
	--function-url-auth-type NONE \
	--region "${REGION}" >/dev/null 2>&1 || true
aws lambda add-permission \
	--function-name "${FUNCTION}" \
	--statement-id FunctionURLAllowInvoke \
	--action lambda:InvokeFunction \
	--principal "*" \
	--region "${REGION}" >/dev/null 2>&1 || true

URL="$(aws lambda get-function-url-config \
	--function-name "${FUNCTION}" \
	--region "${REGION}" \
	--query FunctionUrl --output text)"

echo
echo "Deployed (pure Go zip)."
echo "  ${URL}"
echo "  ${URL}health"
echo "  ${URL}durl/derobeats.tela"
echo "  ${URL}scid/b1e1cba50cbfd8edbb12b01220ffebbece300d4936516a87fc2255fa8e23d8a2/"
echo
echo "Note: first request can take 1–3+ minutes (cold start + chain clone)."

rm -f "${ENV_FILE}"
