# AWS Terraform-First Plan

This document tracks what is managed in Terraform versus what is still manual/external.

## Terraform-managed AWS acquisition

### Foundation
- S3 + DynamoDB Terraform backend state/locking.
- VPC, subnets, route tables, optional NAT.
- Security groups for frontend/api/internal tiers.
- KMS key and alias.
- CloudWatch log groups and retention policies.
- Secrets Manager placeholder secret.

### Identity and access
- Cognito user pool, app client, hosted domain.
- IAM backend/deploy-runner roles and policies.
- Optional Terraform-managed GitHub OIDC provider.
- Optional Terraform-managed GitHub deploy role trust and permissions.

### Runtime and delivery
- ECR repositories and lifecycle policies.
- ECS control-plane cluster and service discovery.
- Optional control-plane services baseline:
  - ALB
  - ALB listener/target group
  - ECS task definitions
  - ECS services (`control-api`, `deploy-orchestrator`, `webhook-ingestor`)
- SQS deploy/webhook queues + DLQs + alarms.
- Static runtime (S3 + CloudFront) + alarm baseline.

### AI and hardening
- AI runtime IAM role/policy.
- AI feature flags and kill switch in SSM.
- AI request log group.
- Optional CloudTrail baseline.
- Optional regional WAF baseline.

### DNS and certificates
- Optional Route53 alias records for API and frontend.
- Optional API ACM certificate + DNS validation.

## Current AWS Resources (Subject to change and update as we add more)
- VPC: `aws_vpc`, `aws_subnet` (public/private), `aws_route_table`, `aws_route_table_association`, optional `aws_nat_gateway`, `aws_internet_gateway`.
- Security perimeter: `aws_security_group` for frontend/API/internal traffic boundaries.
- Identity and auth: `aws_cognito_user_pool`, `aws_cognito_user_pool_client`, optional `aws_cognito_user_pool_domain`.
- IAM: `aws_iam_role`, `aws_iam_policy`, `aws_iam_role_policy_attachment`, optional `aws_iam_openid_connect_provider` for GitHub OIDC.
- Encryption and secrets: `aws_kms_key`, `aws_kms_alias`, `aws_secretsmanager_secret`.
- Observability: `aws_cloudwatch_log_group`, `aws_cloudwatch_metric_alarm`.
- Messaging and async: `aws_sqs_queue` for deploy/webhook queues + DLQs.
- Containers and compute: `aws_ecs_cluster`, `aws_ecs_task_definition`, `aws_ecs_service`, `aws_service_discovery_private_dns_namespace`.
- Load balancing: `aws_lb`, `aws_lb_listener`, `aws_lb_target_group` (when control-plane services baseline is enabled).
- Container registry: `aws_ecr_repository`, `aws_ecr_lifecycle_policy`.
- Static delivery: `aws_s3_bucket` (artifact/static hosting), `aws_cloudfront_distribution`, related bucket policy and access controls.
- AI runtime controls: `aws_ssm_parameter` (feature flags / kill switch), AI runtime IAM resources, AI log group.
- Optional metadata host: `aws_instance`, `aws_iam_instance_profile` (+ optional SSM EC2 role attachments).
- Optional governance: `aws_cloudtrail`, CloudTrail S3 bucket and policy, optional `aws_wafv2_web_acl` + association.
- Optional DNS/certs: `aws_route53_record`, optional `aws_acm_certificate`, `aws_route53_record` validation, `aws_acm_certificate_validation`.

## Manual or external steps (when applicable)
- Customer-owned cross-account role creation in customer AWS accounts.
- GitHub-side setup (repo app/webhook wiring, branch protections, PR rules).
- Domain registrar NS delegation when not using Route53 as authoritative DNS.
- CloudFront custom ACM certificate in `us-east-1` if not using a dedicated Terraform provider alias/stack.

## Customer one-click onboarding assets
- CloudFormation template:
  `labra-infra/customer-onboarding/customer-assume-role.cfn.yaml`
- Usage guide:
  `README.md#customer-onboarding-assumerole-one-click`

The template creates the customer-side IAM role and trust policy with
`sts:ExternalId` protection, so customers do not need to configure trust manually
in the AWS Console.

## Recommended `terraform.tfvars` rollout

1. Baseline (safe default):
- Keep `enable_foundation_modules=true`, `enable_cognito_baseline=true`, `enable_control_plane_cluster=true`, `enable_ecr_baseline=true`.

2. Enable full platform acquisition in Terraform:
- `enable_control_plane_services_baseline=true`
- `enable_metadata_host_baseline=true`
- `enable_cloudtrail_baseline=true`
- `enable_waf_regional_baseline=true`

3. Enable DNS + cert automation (if Route53 zone exists):
- `enable_edge_dns_baseline=true`
- `edge_dns_hosted_zone_id=<zone-id>`
- `api_domain_name=api.<domain>`
- `edge_dns_create_api_certificate=true`
- `frontend_domain_name=<app-domain>`

4. Eliminate manual AWS OIDC setup:
- `iam_create_github_oidc_provider=true`
- `iam_enable_github_oidc_role=true`
- `iam_github_repository=<owner/repo>`

## Detailed go-live checklist (Terraform-first)

1. Bootstrap Terraform state once (platform account):
- `terraform -chdir=labra-infra/env/dev init -backend=false`
- `AWS_EC2_METADATA_DISABLED=true terraform -chdir=labra-infra/env/dev plan -input=false -lock=false -refresh=false -var='bootstrap_state_backend=true'`
- `AWS_EC2_METADATA_DISABLED=true terraform -chdir=labra-infra/env/dev apply -input=false -auto-approve -var='bootstrap_state_backend=true'`

2. Switch to remote backend:
- Copy `labra-infra/env/dev/backend.hcl.example` to `labra-infra/env/dev/backend.hcl`.
- Fill backend values (state bucket/table/region).
- `terraform -chdir=labra-infra/env/dev init -reconfigure -backend-config=backend.hcl`

3. Enable Terraform-managed modules in `labra-infra/env/dev/terraform.tfvars`:
- Keep baseline enabled:
  `enable_foundation_modules=true`
  `enable_control_plane_cluster=true`
  `enable_ecr_baseline=true`
  `enable_deployment_messaging=true`
  `enable_ai_runtime_baseline=true`
- Add optional full management:
  `enable_control_plane_services_baseline=true`
  `enable_metadata_host_baseline=true`
  `enable_cloudtrail_baseline=true`
  `enable_waf_regional_baseline=true`
  `enable_edge_dns_baseline=true` (when hosted zone exists)
- Reduce manual IAM setup:
  `iam_create_github_oidc_provider=true`
  `iam_enable_github_oidc_role=true`
  `iam_github_repository=<owner/repo>`

4. Plan and apply platform infra:
- `AWS_EC2_METADATA_DISABLED=true terraform -chdir=labra-infra/env/dev plan -input=false -lock=false -refresh=false`
- `AWS_EC2_METADATA_DISABLED=true terraform -chdir=labra-infra/env/dev apply -input=false`

5. Capture critical Terraform outputs for runtime configuration:
- Platform role ARN for customer trust:
  `terraform -chdir=labra-infra/env/dev output -raw backend_service_role_arn`
- Cognito values (if enabled):
  `terraform -chdir=labra-infra/env/dev output -raw cognito_user_pool_id`
  `terraform -chdir=labra-infra/env/dev output -raw cognito_app_client_id`
- Static site/edge values:
  `terraform -chdir=labra-infra/env/dev output -raw static_site_url`
  `terraform -chdir=labra-infra/env/dev output -raw static_distribution_domain_name`

6. Onboard each customer account with one-click role creation:
- Use CloudFormation one-command deploy from:
  `README.md#customer-onboarding-assumerole-one-click`
- Provide customers:
  platform role ARN from step 5, plus a generated external ID.
- Customer returns role ARN; store in Labra AWS connection settings with region and external ID.

7. Configure GitHub integration manually (GitHub-side):
- Install/configure GitHub App or OAuth app.
- Configure webhook secret and webhook endpoint.
- Configure branch protections / PR rules.

8. Configure DNS/cert edge cases manually only when needed:
- If registrar is external: update NS delegation manually.
- If using CloudFront custom cert in `us-east-1` without provider alias stack:
  issue/import cert manually or in separate Terraform stack.

9. Validate operational readiness:
- Backend: `go test ./... && go vet ./...`
- Frontend: `npm run check && node --test tests/smoke.route-map.test.mjs tests/component.shell.test.mjs && npm run build`
- Infra: `AWS_EC2_METADATA_DISABLED=true terraform -chdir=labra-infra/env/dev validate`

10. Start the app stack locally after infra configuration:
- Backend env (`labra-backend/.env`) minimum:
  `DB_URL=./labra.db`
  `APP_ENV=dev`
  `API_HOST=localhost`
  `API_PORT=8080`
  optional auth/webhook values:
  `JWT_ISSUER`, `JWT_AUDIENCE`, `JWT_SIGNING_SECRET`, `GITHUB_WEBHOOK_SECRET`, `GH_CLIENT_ID`, `GH_CLIENT_SECRET`
- Start backend:
  `cd labra-backend && go run ./cmd`
- Start frontend:
  `cd labra-frontend && npm run dev`
- Open UI:
  `http://localhost:5173`
