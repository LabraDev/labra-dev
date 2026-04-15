# labra-infra

Sprint 1 infra baseline now includes:
- `state-bootstrap` (S3 + DynamoDB)
- `vpc-baseline` (VPC, subnets, route tables, optional NAT)
- `security-groups-baseline` (frontend/api/internal)
- `iam-baseline` (backend role, deploy-runner role, optional GitHub OIDC role)
- `kms-baseline` (encryption key + alias)
- `logging-baseline` (CloudWatch log groups + retention)
- `secrets-baseline` (Secrets Manager envelope)
- `static_runtime` (S3 + CloudFront static hosting)

## Files we use most
- `env/dev/main.tf` composition
- `env/dev/variables.tf` inputs for this env
- `env/dev/terraform.tfvars` dev values
- `env/dev/outputs.tf` outputs backend/frontend should read
- `env/dev/backend.hcl.example` backend bootstrap template

## Team workflow
1. Bootstrap backend once
   - set `bootstrap_state_backend = true`
   - run `terraform init` and `terraform apply` from `env/dev`
2. Move to remote backend
   - copy `backend.hcl.example` to `backend.hcl`
   - run `terraform init -reconfigure -backend-config=backend.hcl`
3. Normal deploy flow
   - set `bootstrap_state_backend = false`
   - run `terraform plan` and `terraform apply`

## Security defaults
- Encryption enabled for Terraform state and static site bucket.
- Public access blocked on S3 buckets.
- IAM roles scoped for backend/deploy runner.
- Optional GitHub OIDC role for CI/CD without long-lived AWS keys.
