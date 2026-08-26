# Deployment

The repository supports the existing EC2 application deployment and the unified Phase 11 serverless deployment.

## Layout

- deployment/serverful: EC2 backend, edge, frontend publication, shared helpers, and infrastructure/tf state.
- deployment/serverless: the only active serverless implementation, split into database, backend, frontend, common orchestration, and one infrastructure/tf state.

## Requirements

- authenticated AWS CLI
- Terraform 1.6 or newer
- Go and pnpm
- uv with Python 3.14 available
- ssh, scp, and ssh-keyscan
- an existing VPC, Internet-Gateway-backed subnet, EC2 key pair, and public Route 53 hosted zone
- Lambda account concurrency quota of at least 4

## EC2 Application

- make deploy-serverful: deploy the EC2-backed application and frontend.
- make deploy-serverful all: apply infrastructure, deploy backend/frontend, and deploy nginx.
- make deploy-serverful infra, make deploy-serverful backend, make deploy-serverful frontend, and make deploy-serverful edge: scoped serverful operations.
- make tf-init, make tf-plan, and make tf-apply: serverful Terraform operations.
- make destroy: destroy the serverful deployment.

## Unified Serverless Deployment

Run make deploy ACTION=init once to create the mode-0600 external JSON template at ~/.config/expense-tracker/serverless.json. DEPLOY_CONFIG_FILE may override that location with another absolute external path.

Normal commands:

- make deploy
- make deploy ACTION=plan
- make deploy ACTION=deploy
- make deploy ACTION=update SCOPE=migrations
- make deploy ACTION=update SCOPE=backend
- make deploy ACTION=update SCOPE=frontend
- make deploy ACTION=update SCOPE=all
- make deploy ACTION=status
- make deploy ACTION=destroy

The JSON file is the sole human-edited serverless deployment source. Terraform receives a generated non-secret projection. PostgreSQL and Lambda runtime values are generated into temporary protected files, published directly, and removed.

Detailed configuration, sequencing, resume behavior, verification, and destroy rules are documented in deployment/serverless/README.md.

## Safety

- Review the saved initial Terraform plan before applying.
- Keep deployment config, SSH keys, tokens, credentials, and operator-owned value files outside version control.
- Never run ordinary Terraform refresh/plan/apply after Lambda runtime publication.
- Serverless destroy deletes Lambdas first, waits 20 minutes plus five additional minutes for owned ENIs, and only then permits explicitly enabled bounded force cleanup.
- Back up required database data before destroy.
