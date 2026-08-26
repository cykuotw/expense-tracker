# Unified Serverless Deployment

This directory is the only Phase 11 serverless deployment implementation. It owns the database host, Worker and Bootstrap Lambdas, API custom domain, and Phase 10 frontend infrastructure/publication through one Python entrypoint and one Terraform state.

## Operator configuration

Create the protected external configuration once:

```bash
make deploy ACTION=init
```

The default path is `~/.config/expense-tracker/serverless.json`. Set `DEPLOY_CONFIG_FILE` to an absolute external path to override it. The file, SSH key, and optional Google token must be regular non-symlink files outside the repository with mode `0600`.

The JSON sections are `deployment`, `aws`, `database`, `backend`, `frontend`, optional `first_admin`, and `local_credentials`. This is the sole human-edited serverless deployment source. Do not create serverless `.tfvars`, PostgreSQL password files, or Lambda runtime JSON files; the deployer creates protected temporary projections and removes them.

## Commands

```bash
make deploy
make deploy ACTION=plan
make deploy ACTION=deploy
make deploy ACTION=update SCOPE=migrations
make deploy ACTION=update SCOPE=backend
make deploy ACTION=update SCOPE=frontend
make deploy ACTION=update SCOPE=all
make deploy ACTION=status
make deploy ACTION=destroy
```

`make deploy` deploys/resumes an incomplete environment and runs migrations → backend → frontend when the environment is complete. Initial infrastructure requires typing the configured name prefix. Destroy requires typing `destroy-<name_prefix>`.

Use `SERVERLESS_AUTO_APPROVE=true` only in a controlled disposable test. `FORCE_DETACH_LAMBDA_ENI=true` permits the bounded owned-ENI force cleanup only after the normal 20-minute wait plus five additional minutes.

## Safety boundary

- Terraform never receives database/JWT/first-admin secrets and never manages Lambda environments.
- The Worker begins at reserved concurrency `0`; Python publishes runtime configuration and activates it at `3`.
- The raw execute-api endpoint is disabled only after custom-domain and frontend checks pass.
- Normal updates do not run Terraform.
- Destroy deletes Lambdas first, waits for their owned ENIs, then runs `terraform destroy -refresh=false`.
- A deployment failure keeps persistent resources for an explicit resume; it never performs automatic rollback or destroy.
