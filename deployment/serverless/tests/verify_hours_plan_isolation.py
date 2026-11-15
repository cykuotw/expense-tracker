"""Check schedule-only targeting against legacy EC2 state and host drift.

Uses synthetic state, fake credentials and plan-only operations. No production
state/configuration, resource applies, or AWS API requests are needed. Both alias
stages and schedule activation states use the production target/validation code.
"""
from pathlib import Path
import json
import shutil
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
from common.terraform import Terraform, plan_actions
from database.hours import TARGETS, validate_plan


def main() -> None:
    installed = ROOT / "infrastructure/tf"
    stub = (Path(__file__).parent / "terraform_hours/stub.tf").read_text()
    stub = stub.replace('region = "ca-central-1"', '''region = "ca-central-1"
  access_key = "synthetic-access-key"
  secret_key = "synthetic-secret-key"
  skip_credentials_validation = true
  skip_requesting_account_id = true
  skip_metadata_api_check = true''')
    # A host replacement is pending, and its v1 schema would require an AWS
    # lookup if the schedule plan accidentally traversed the host dependency.
    stub = stub.replace('ami           = "ami-00000000000000000"', 'ami           = "ami-11111111111111111"')
    state = {
        "version": 4, "terraform_version": "1.14.7", "serial": 1,
        "lineage": "00000000-0000-0000-0000-000000000001", "outputs": {},
        "resources": [{
            "mode": "managed", "type": "aws_instance", "name": "postgres",
            "provider": 'provider["registry.terraform.io/hashicorp/aws"]',
            "instances": [{"schema_version": 1, "attributes": {
                "id": "i-00000000000000000", "ami": "ami-00000000000000000", "instance_type": "t4g.micro",
                "launch_template": [{"id": "lt-00000000000000000", "name": "synthetic-existing", "version": "1"}],
            }, "sensitive_attributes": []}],
        }],
    }
    with tempfile.TemporaryDirectory(prefix="expense-hours-isolation-") as temporary:
        work = Path(temporary)
        (work / "stub.tf").write_text(stub)
        (work / "terraform.tfstate").write_text(json.dumps(state))
        before = (work / "terraform.tfstate").read_bytes()
        shutil.copy(installed / "operating_hours.tf", work / "operating_hours.tf")
        shutil.copy(installed / ".terraform.lock.hcl", work / ".terraform.lock.hcl")
        (work / ".terraform").mkdir()
        (work / ".terraform/providers").symlink_to(installed / ".terraform/providers", target_is_directory=True)
        subprocess.run(["terraform", f"-chdir={work}", "init", "-backend=false", "-input=false"], check=True, capture_output=True)
        for aliases in (False, True):
            for active in (False, True):
                variables = work / "variables.json"
                variables.write_text(json.dumps({
                    "operating_hours": {"configured": True, "enabled": True, "start": "09:00", "stop": "22:00", "safety_stop": "01:00", "timezone": "America/Toronto"},
                    "operating_hours_database_instance_id": "i-00000000000000000",
                    "use_lambda_aliases": aliases, "activate_operating_hours": active,
                }))
                client = Terraform(work, variables)
                plan = work / "hours.tfplan"
                client.plan(plan, targets=TARGETS)
                planned = client.show_plan(plan)
                all_actions = plan_actions(planned)
                # The fixture intentionally has no pre-existing application
                # resources. Account explicitly for only those three creates;
                # a real hours-apply still uses the unmodified strict guard.
                fixture_dependencies = {"aws_iam_role.sender", "aws_lambda_function.delivery", "aws_s3_bucket.frontend"}
                if any(all_actions.get(key) != ["create"] for key in fixture_dependencies):
                    raise RuntimeError("fixture application prerequisites changed")
                schedule_plan = {"resource_changes": [item for item in planned.get("resource_changes", []) if item["address"] not in fixture_dependencies]}
                actions = validate_plan(schedule_plan, enabled=True)
                if 'aws_scheduler_schedule.database_hours["start"]' not in actions:
                    raise RuntimeError("fixture did not exercise schedule planning")
                if any(key in actions for key in ("aws_instance.postgres", "aws_launch_template.postgres")):
                    raise RuntimeError("schedule planning reached database resources")
                if (work / "terraform.tfstate").read_bytes() != before:
                    raise RuntimeError("planning changed fixture state")
                print(f"hours_plan_isolation aliases={aliases} active={active} passed state_unchanged=true")


if __name__ == "__main__":
    main()
