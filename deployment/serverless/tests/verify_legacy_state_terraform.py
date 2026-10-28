"""Exercise targeted-plan inspection with synthetic v5 EC2 state and provider v6.

Run through uv after initializing the serverless provider. No production state,
deployment configuration, AWS requests, or resource applies are used.
"""
from pathlib import Path
import json
import shutil
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
from common.terraform import Terraform, require_create_only


CONFIG = '''
terraform {
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 6.23" }
  }
}
provider "aws" {
  region                      = "ca-central-1"
  access_key                  = "synthetic-access-key"
  secret_key                  = "synthetic-secret-key"
  skip_credentials_validation = true
  skip_requesting_account_id  = true
  skip_metadata_api_check     = true
}
variable "use_lambda_aliases" { type = bool }
resource "aws_instance" "postgres" {
  ami           = "ami-00000000000000000"
  instance_type = "t4g.micro"
}
resource "aws_cloudwatch_log_group" "frontend" {
  name = var.use_lambda_aliases ? "synthetic-frontend-live" : "synthetic-frontend"
}
'''


def main() -> None:
    installed = ROOT / "infrastructure/tf"
    with tempfile.TemporaryDirectory(prefix="expense-legacy-state-") as directory:
        work = Path(directory)
        (work / "main.tf").write_text(CONFIG)
        shutil.copy(installed / ".terraform.lock.hcl", work / ".terraform.lock.hcl")
        (work / ".terraform").mkdir()
        (work / ".terraform/providers").symlink_to(installed / ".terraform/providers", target_is_directory=True)
        state = {
            "version": 4, "terraform_version": "1.15.8", "serial": 1,
            "lineage": "00000000-0000-0000-0000-000000000001", "outputs": {},
            "resources": [{
                "mode": "managed", "type": "aws_instance", "name": "postgres",
                "provider": 'provider["registry.terraform.io/hashicorp/aws"]',
                "instances": [{"schema_version": 1, "attributes": {
                    "id": "i-00000000000000000", "ami": "ami-00000000000000000", "instance_type": "t4g.micro",
                }, "sensitive_attributes": []}],
            }],
        }
        state_path = work / "terraform.tfstate"
        state_path.write_text(json.dumps(state))
        before = state_path.read_bytes()
        subprocess.run(["terraform", f"-chdir={work}", "init", "-backend=false", "-input=false"], check=True, capture_output=True)
        for aliases in (False, True):
            variables = work / "variables.json"
            variables.write_text(json.dumps({"use_lambda_aliases": aliases}))
            client = Terraform(work, variables)
            plan = work / "target.tfplan"
            client.plan(plan, targets=("aws_cloudwatch_log_group.frontend",))
            old_inspection = subprocess.run(["terraform", f"-chdir={work}", "show", "-json", str(plan)], capture_output=True, text=True)
            if old_inspection.returncode == 0 or "schema version 1 for aws_instance.postgres" not in old_inspection.stderr:
                raise RuntimeError("legacy-state fixture no longer reproduces the JSON schema failure")
            actions = require_create_only(client.show_plan(plan))
            if actions != {"aws_cloudwatch_log_group.frontend": ["create"]}:
                raise RuntimeError(f"target scope changed: {actions}")
            if state_path.read_bytes() != before:
                raise RuntimeError("inspection modified synthetic state")
            print(f"legacy_state_targeted_plan aliases={aliases} passed state_unchanged=true")


if __name__ == "__main__":
    main()
