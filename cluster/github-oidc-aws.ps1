# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
#
# Lets GitHub Actions deploy to EKS with short-lived OIDC credentials (no access keys in GitHub).
# Safe to run again: anything that already exists is kept.
#
# Usage:  .\cluster\github-oidc-aws.ps1 -GitHubRepo "your-github-user/50-MicroServices-Project-By-DevOpsShack"
#   GitHub's owner and repository IDs are looked up automatically: nothing else to type.
param(
    [Parameter(Mandatory = $true)][string]$GitHubRepo,
    [string]$OwnerId = "",       # GitHub's numeric owner ID; looked up automatically when empty
    [string]$RepoId = "",        # GitHub's numeric repository ID; looked up automatically when empty
    [string]$Cluster = "railshack",
    [string]$Region = "ap-south-1",
    [string]$Role = "railshack-github-deployer"
)
$ErrorActionPreference = "Continue"   # we check every aws call's exit code ourselves

function Run-Aws {   # run the AWS CLI, stop with a clear message if it fails
    $out = & aws @args 2>&1
    if ($LASTEXITCODE -ne 0) { Write-Host ($out | Out-String) -ForegroundColor Red; throw "aws $($args[0..1] -join ' ') failed" }
    $out
}

$account = (& aws sts get-caller-identity --query Account --output text 2>$null | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or $account -notmatch '^\d{12}$') { throw "The AWS CLI is not logged in (aws sts get-caller-identity failed)" }
$issuer = "token.actions.githubusercontent.com"
$tmp = [IO.Path]::GetTempPath()

# 1. Trust GitHub's OIDC issuer (once per AWS account).
& aws iam get-open-id-connect-provider --open-id-connect-provider-arn "arn:aws:iam::${account}:oidc-provider/$issuer" *> $null
if ($LASTEXITCODE -eq 0) { Write-Host "OIDC provider already exists" }
else { Run-Aws iam create-open-id-connect-provider --url "https://$issuer" --client-id-list sts.amazonaws.com | Out-Null; Write-Host "OIDC provider created" }

# 2. Work out the subject GitHub puts in its token. Nothing to type: the IDs are looked up here.
#    Repositories created after 15 July 2026 use GitHub's immutable format with numeric IDs:
#      repo:<owner>@<owner-id>/<repo>@<repo-id>:environment:production-eks
#    Older repositories use repo:<owner>/<repo>:environment:production-eks. We trust both.
#    The owner ID is public, so anyone can look it up. The repository ID of a private repo is not:
#    when it can't be read, we match any repository ID, still only for this owner ID and this repo name.
$owner, $repoName = $GitHubRepo.Split("/")
function GitHub-Id([string]$path) {
    if (Get-Command gh -ErrorAction SilentlyContinue) {
        $id = (& gh api $path --jq .id 2>$null | Out-String).Trim()
        if ($LASTEXITCODE -eq 0 -and $id -match '^\d+$') { return $id }
    }
    try { return [string](Invoke-RestMethod "https://api.github.com/$path" -Headers @{ "User-Agent" = "railshack" }).id } catch { return "" }
}
if (-not $OwnerId) { $OwnerId = GitHub-Id "users/$owner" }
if (-not $OwnerId) { throw "Could not reach api.github.com to look up the ID of '$owner'. Check the internet connection and the -GitHubRepo spelling." }
if (-not $RepoId)  { $RepoId  = GitHub-Id "repos/$GitHubRepo" }
$repoPart = if ($RepoId) { $RepoId } else { "*" }
$subImmutable = "repo:${owner}@${OwnerId}/${repoName}@${repoPart}:environment:production-eks"
$subClassic   = "repo:${GitHubRepo}:environment:production-eks"
$subs = @($subImmutable, $subClassic)
$condition = [ordered]@{ StringEquals = [ordered]@{ "${issuer}:aud" = "sts.amazonaws.com" } }
if ($RepoId) { $condition.StringEquals["${issuer}:sub"] = $subs }      # exact match
else         { $condition["StringLike"] = @{ "${issuer}:sub" = $subs } } # "*" in place of the private repo ID
Write-Host "Trusting: $subImmutable"

# 3. A role only the production-eks environment of this repository can assume.
$trust = [ordered]@{
    Version   = "2012-10-17"
    Statement = @([ordered]@{
        Effect    = "Allow"
        Principal = @{ Federated = "arn:aws:iam::${account}:oidc-provider/$issuer" }
        Action    = "sts:AssumeRoleWithWebIdentity"
        Condition = $condition
    })
} | ConvertTo-Json -Depth 10
$policy = '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["eks:DescribeCluster"],"Resource":"*"}]}'
$trustFile = Join-Path $tmp "railshack-trust.json"
$policyFile = Join-Path $tmp "railshack-policy.json"
[IO.File]::WriteAllText($trustFile, $trust)      # UTF-8 without BOM, which the AWS CLI expects
[IO.File]::WriteAllText($policyFile, $policy)

& aws iam get-role --role-name $Role *> $null
if ($LASTEXITCODE -eq 0) {
    Run-Aws iam update-assume-role-policy --role-name $Role --policy-document "file://$trustFile" | Out-Null
    Write-Host "Role $Role already exists: trust policy updated"
} else {
    Run-Aws iam create-role --role-name $Role --assume-role-policy-document "file://$trustFile" --output text | Out-Null
    Write-Host "Role $Role created"
}
Run-Aws iam put-role-policy --role-name $Role --policy-name eks-describe --policy-document "file://$policyFile" | Out-Null

# 4. Give that role admin access inside the cluster (EKS access entries).
$roleArn = "arn:aws:iam::${account}:role/$Role"
& aws eks describe-access-entry --cluster-name $Cluster --region $Region --principal-arn $roleArn *> $null
if ($LASTEXITCODE -ne 0) {
    # A brand-new IAM role takes a few seconds to become visible to EKS ("invalid principal"), so retry.
    $created = $false
    foreach ($try in 1..12) {
        $out = & aws eks create-access-entry --cluster-name $Cluster --region $Region --principal-arn $roleArn --output text 2>&1
        if ($LASTEXITCODE -eq 0) { $created = $true; break }
        if (($out | Out-String) -notmatch "invalid principal") { Write-Host ($out | Out-String) -ForegroundColor Red; throw "aws eks create-access-entry failed" }
        Write-Host "Waiting for the new role to reach EKS ($try/12)..."
        Start-Sleep -Seconds 10
    }
    if (-not $created) { throw "EKS still does not see $roleArn after 2 minutes: run this script again" }
}
Write-Host "EKS access entry ready"
Run-Aws eks associate-access-policy --cluster-name $Cluster --region $Region --principal-arn $roleArn `
    --policy-arn arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy --access-scope type=cluster --output text | Out-Null

Remove-Item $trustFile, $policyFile -ErrorAction SilentlyContinue
Write-Host ""
Write-Host "Add this as the AWS_ROLE_ARN secret in GitHub: $roleArn" -ForegroundColor Green
