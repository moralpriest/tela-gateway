#!/usr/bin/env bash
# Deploys the TELA INDEX indexer as an AWS Lambda (x86_64) that scans the DERO
# chain weekly and writes a fresh aliases.json to S3. The gateway Lambda reads
# that object on cold start via ALIASES_S3_URI.
#
# Usage: scripts/deploy-indexer-lambda.sh [--build-only]
set -euo pipefail

cd "$(dirname "$0")/.."

AWS_REGION="${AWS_REGION:-us-east-1}"
ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
BUCKET="cypher-punks-tela-aliases"
FUNCTION="tela-indexer"
ROLE_NAME="tela-indexer-lambda-role"
BINARY="bin/tela-indexer"
ZIP="bin/tela-indexer.zip"

DERO_DAEMON_URLS="${DERO_DAEMON_URLS:-node.derofoundation.org:11012,dero.rabidmining.com:10102,dero-node.net:10102,community-pools.mysrv.cloud:10102}"

echo "==> building indexer (linux/amd64)"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "$BINARY" ./cmd/indexer
mkdir -p bin
zip -j "$ZIP" "$BINARY"

if [[ "${1:-}" == "--build-only" ]]; then
  echo "built $ZIP"
  exit 0
fi

echo "==> ensuring S3 bucket $BUCKET"
if ! aws s3api head-bucket --bucket "$BUCKET" 2>/dev/null; then
  aws s3 mb "s3://$BUCKET" --region "$AWS_REGION"
fi

echo "==> ensuring IAM role $ROLE_NAME"
if ! aws iam get-role --role-name "$ROLE_NAME" >/dev/null 2>&1; then
  TRUST='{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}'
  aws iam create-role --role-name "$ROLE_NAME" --assume-role-policy-document "$TRUST" >/dev/null
fi
aws iam attach-role-policy --role-name "$ROLE_NAME" --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole >/dev/null 2>&1 || true
# S3 write access for the aliases bucket.
CAT='{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:PutObject","s3:GetObject"],"Resource":"arn:aws:s3:::'$BUCKET'/*"}]}'
aws iam put-role-policy --role-name "$ROLE_NAME" --policy-name tela-indexer-s3 --policy-document "$CAT" >/dev/null

sleep 8
ROLE_ARN="$(aws iam get-role --role-name "$ROLE_NAME" --query Role.Arn --output text)"

if aws lambda get-function --function-name "$FUNCTION" >/dev/null 2>&1; then
  echo "==> updating $FUNCTION"
  aws lambda update-function-code --function-name "$FUNCTION" --zip-file "fileb://$ZIP" --region "$AWS_REGION" >/dev/null
  aws lambda wait function-updated --function-name "$FUNCTION" --region "$AWS_REGION"
  aws lambda update-function-configuration --function-name "$FUNCTION" \
    --runtime provided.al2023 --handler bootstrap \
    --architectures x86_64 --memory-size 2048 --timeout 900 \
    --region "$AWS_REGION" \
    --environment "Variables={DERO_DAEMON_URLS=$DERO_DAEMON_URLS,ALIASES_S3_URI=s3://$BUCKET/aliases.json,ALIASES_OUT=/tmp/aliases.json}" >/dev/null
else
  echo "==> creating $FUNCTION"
  aws lambda create-function --function-name "$FUNCTION" \
    --runtime provided.al2023 --handler bootstrap --zip-file "fileb://$ZIP" \
    --role "$ROLE_ARN" --architectures x86_64 --memory-size 2048 --timeout 900 \
    --region "$AWS_REGION" \
    --environment "Variables={DERO_DAEMON_URLS=$DERO_DAEMON_URLS,ALIASES_S3_URI=s3://$BUCKET/aliases.json,ALIASES_OUT=/tmp/aliases.json}" >/dev/null
fi

echo "==> scheduling weekly EventBridge trigger (Mon 06:00 UTC)"
RULE="tela-indexer-weekly"
aws events put-rule --name "$RULE" --schedule-expression 'cron(0 6 ? * MON *)' --region "$AWS_REGION" >/dev/null
aws lambda add-permission --function-name "$FUNCTION" --statement-id "$RULE" \
  --action lambda:InvokeFunction --principal events.amazonaws.com \
  --source-arn "arn:aws:events:$AWS_REGION:$ACCOUNT_ID:rule/$RULE" --region "$AWS_REGION" >/dev/null 2>&1 || true
TARGETS='[{"Id":"1","Arn":"'"$(aws lambda get-function --function-name "$FUNCTION" --region "$AWS_REGION" --query Configuration.FunctionArn --output text)"'"}]'
aws events put-targets --rule "$RULE" --targets "$TARGETS" --region "$AWS_REGION" >/dev/null

echo "done. indexer Lambda: $FUNCTION | bucket: s3://$BUCKET/aliases.json"
