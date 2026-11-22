#!/usr/bin/env bash
# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
# Lets GitHub Actions deploy to EKS with short-lived OIDC credentials (no access keys in GitHub).
# Usage (Linux/macOS): GITHUB_REPO=<owner>/<repo> cluster/github-oidc-aws.sh
# On Windows use the PowerShell version: .\cluster\github-oidc-aws.ps1 -GitHubRepo "<owner>/<repo>"
set -euo pipefail
: "${GITHUB_REPO:?set GITHUB_REPO=owner/repo}"
CLUSTER="${CLUSTER:-railshack}"
REGION="${REGION:-ap-south-1}"
ROLE="${ROLE:-railshack-github-deployer}"
ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
# Repositories created after 15 July 2026 use GitHub's immutable subject: repo:<owner>@<owner-id>/<repo>@<repo-id>:...
OWNER="${GITHUB_REPO%%/*}"; NAME="${GITHUB_REPO#*/}"
OWNER_ID="${OWNER_ID:-$(gh api "users/$OWNER" --jq .id 2>/dev/null || curl -fsS "https://api.github.com/users/$OWNER" | python3 -c 'import json,sys;print(json.load(sys.stdin)["id"])')}"
REPO_ID="${REPO_ID:-$(gh api "repos/$GITHUB_REPO" --jq .id 2>/dev/null || curl -fsS "https://api.github.com/repos/$GITHUB_REPO" 2>/dev/null | python3 -c 'import json,sys;print(json.load(sys.stdin)["id"])' 2>/dev/null || true)}"
: "${OWNER_ID:?could not reach api.github.com to look up the owner ID}"
# A private repo's ID can't be read without logging in: then match any repo ID for this owner ID and repo name.
MATCH=StringLike; [ -n "$REPO_ID" ] || REPO_ID="*"   # StringLike without a "*" is an exact match

# 1. Trust GitHub's OIDC issuer (once per AWS account; ignore "already exists").
aws iam create-open-id-connect-provider --url https://token.actions.githubusercontent.com \
  --client-id-list sts.amazonaws.com 2>/dev/null || echo "OIDC provider already exists"

# 2. A role only the production-eks environment of this repo can assume.
cat > /tmp/trust.json <<JSON
{ "Version": "2012-10-17",
  "Statement": [{ "Effect": "Allow",
    "Principal": { "Federated": "arn:aws:iam::${ACCOUNT}:oidc-provider/token.actions.githubusercontent.com" },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": { "token.actions.githubusercontent.com:aud": "sts.amazonaws.com" },
      "${MATCH}": { "token.actions.githubusercontent.com:sub": [ "repo:${OWNER}@${OWNER_ID}/${NAME}@${REPO_ID}:environment:production-eks",
                                                                    "repo:${GITHUB_REPO}:environment:production-eks" ] } } }] }
JSON
aws iam create-role --role-name "$ROLE" --assume-role-policy-document file:///tmp/trust.json -o none
aws iam put-role-policy --role-name "$ROLE" --policy-name eks-describe \
  --policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["eks:DescribeCluster"],"Resource":"*"}]}'

# 3. Give that role admin access inside the cluster (EKS access entries).
ROLE_ARN="arn:aws:iam::${ACCOUNT}:role/${ROLE}"
aws eks create-access-entry --cluster-name "$CLUSTER" --region "$REGION" --principal-arn "$ROLE_ARN" -o none
aws eks associate-access-policy --cluster-name "$CLUSTER" --region "$REGION" --principal-arn "$ROLE_ARN" \
  --policy-arn arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy --access-scope type=cluster -o none
echo "✅ Add this as the AWS_ROLE_ARN secret in GitHub: $ROLE_ARN"
