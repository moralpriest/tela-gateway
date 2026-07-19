#!/usr/bin/env bash
# Provisions AWS-native custom-domain serving for the tela-gateway Lambda:
#   1. ACM certificate (us-east-1) for *.tela.cypher-punks.com
#   2. CloudFront distribution -> Lambda Function URL origin
#      - forwards X-Forwarded-Host (Function URL only accepts its own Host)
#      - viewer HTTPS only
#      - cache disabled (TELA apps are dynamic)
#      - aliases: *.tela.cypher-punks.com
# DNS (CNAME for *.tela.cypher-punks.com) stays at the current registrar and
# must point at the CloudFront domain name printed below.
#
# Manual steps this script does NOT perform:
#   - Approving the ACM certificate (email/DNS validation) in the console.
#   - Creating the registrar CNAME/A records.
#
# Usage: scripts/deploy-cloudfront.sh
set -euo pipefail

AWS_REGION="${AWS_REGION:-us-east-1}"
DIST_NAME="tela-gateway"
CERT_ARN_FILE=".cloudfront-cert-arn"
FUNCTION_URL="https://mxp2n34gow7sr53uzoqkin6jge0ziava.lambda-url.us-east-1.on.aws/"

echo "==> requesting ACM certificate (us-east-1)"
if [[ -f "$CERT_ARN_FILE" ]]; then
  CERT_ARN="$(cat "$CERT_ARN_FILE")"
  echo "using existing cert $CERT_ARN"
else
  CERT_ARN="$(aws acm request-certificate \
    --domain-name '*.tela.cypher-punks.com' \
    --validation-method DNS --region us-east-1 \
    --query CertificateArn --output text)"
  echo "$CERT_ARN" > "$CERT_ARN_FILE"
  echo "requested cert $CERT_ARN — approve via DNS validation in the ACM console, then re-run."
  exit 0
fi

# Wait until the cert is issued (after the user validates it).
echo "==> waiting for certificate to be ISSUED"
aws acm wait certificate-validated --certificate-arn "$CERT_ARN" --region us-east-1

ORIGIN_ID="tela-lambda-url"
read -r ODOMAIN OPATH <<< "$(echo "$FUNCTION_URL" | sed -E 's#https://##; s#/##; s#(^[^/]+)/(.*)$#\1 \2#')"
OPATH="/${OPATH}"

# CloudFront strips the viewer Host (Function URLs 403 a mismatched Host), so a
# CloudFront Function copies the viewer Host into X-Forwarded-Host, which the
# gateway reads to route <app>.tela.cypher-punks.com -> the right SCID.
CF_FN_NAME="tela-host-rewrite"
CF_FN_SRC="$(dirname "$0")/cf-function-host-rewrite.js"
echo "==> creating/updating CloudFront Function $CF_FN_NAME"
if aws cloudfront describe-function --name "$CF_FN_NAME" >/dev/null 2>&1; then
  FN_ETAG="$(aws cloudfront describe-function --name "$CF_FN_NAME" --query ETag --output text)"
  aws cloudfront update-function --name "$CF_FN_NAME" --if-match "$FN_ETAG" \
    --function-config "Comment=inject X-Forwarded-Host from viewer Host,Runtime=cloudfront-js-2.0" \
    --function-code "fileb://$CF_FN_SRC" >/dev/null
else
  aws cloudfront create-function --name "$CF_FN_NAME" \
    --function-config "Comment=inject X-Forwarded-Host from viewer Host,Runtime=cloudfront-js-2.0" \
    --function-code "fileb://$CF_FN_SRC" >/dev/null
fi
FN_ETAG="$(aws cloudfront describe-function --name "$CF_FN_NAME" --query ETag --output text)"
aws cloudfront publish-function --name "$CF_FN_NAME" --if-match "$FN_ETAG" >/dev/null
CF_FN_ARN="$(aws cloudfront describe-function --name "$CF_FN_NAME" \
  --query 'FunctionSummary.FunctionMetadata.FunctionARN' --output text)"
echo "    function ARN: $CF_FN_ARN"

echo "==> creating CloudFront distribution"
CONFIG=$(cat <<EOF
{
  "CallerReference": "$DIST_NAME-$(date +%s)",
  "Aliases": { "Quantity": 1, "Items": ["*.tela.cypher-punks.com"] },
  "Origins": {
    "Quantity": 1,
    "Items": [{
      "Id": "$ORIGIN_ID",
      "DomainName": "$ODOMAIN",
      "OriginPath": "",
      "CustomOriginConfig": {
        "HTTPPort": 80, "HTTPSPort": 443,
        "OriginProtocolPolicy": "https-only",
        "OriginSslProtocols": { "Quantity": 1, "Items": ["TLSv1.2"] }
      }
    }]
  },
  "DefaultCacheBehavior": {
    "TargetOriginId": "$ORIGIN_ID",
    "ViewerProtocolPolicy": "redirect-to-https",
    "ForwardedValues": {
      "QueryString": true,
      "Headers": {
        "Quantity": 2,
        "Items": ["X-Forwarded-Host", "X-Forwarded-Proto"]
      },
      "Cookies": { "Forward": "none" }
    },
    "FunctionAssociations": {
      "Quantity": 1,
      "Items": [{ "FunctionARN": "$CF_FN_ARN", "EventType": "viewer-request" }]
    },
    "MinTTL": 0, "DefaultTTL": 0, "MaxTTL": 0
  },
  "Comment": "tela-gateway custom domain",
  "Enabled": true,
  "ViewerCertificate": {
    "ACMCertificateArn": "$CERT_ARN",
    "SSLSupportMethod": "sni-only",
    "MinimumProtocolVersion": "TLSv1.2_2021"
  }
}
EOF
)

DIST_ID="$(aws cloudfront create-distribution --distribution-config "$CONFIG" \
  --query 'Distribution.Id' --output text)"
DOMAIN="$(aws cloudfront get-distribution --id "$DIST_ID" \
  --query 'Distribution.DomainName' --output text)"

echo "CloudFront distribution: $DIST_ID"
echo "CloudFront domain:       $DOMAIN"
echo
echo "At your registrar create:"
echo "  CNAME  *.tela.cypher-punks.com  ->  $DOMAIN"
