state_bucket_name     = "labra-infra-dev-tfstate-974646089985"
state_lock_table_name = "labra-infra-dev-platform-terraform-locks"

roadmap_phase   = "Phase 2"
roadmap_version = "Ver 0.3"
cost_center     = "cpsc465"

enable_foundation_modules    = true
vpc_enable_nat_gateway       = false
enable_cognito_baseline      = true
enable_control_plane_cluster = true

# Enable after creating an IAM OIDC provider in your AWS account.
iam_enable_github_oidc_role = false
# iam_github_oidc_provider_arn = "arn:aws:iam::<account-id>:oidc-provider/token.actions.githubusercontent.com"
# iam_github_repository = "owner/repo"
