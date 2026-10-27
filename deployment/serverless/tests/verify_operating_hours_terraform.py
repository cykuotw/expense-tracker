"""Run isolated mock-provider lifecycle checks; never reads deployment state/config.

Run through uv after the serverless Terraform provider has been initialized.
Only synthetic fixture resources are applied, using Terraform's mock provider.
"""
from pathlib import Path
import shutil
import subprocess
import tempfile


def main() -> None:
    root = Path(__file__).resolve().parents[1] / "infrastructure/tf"
    fixtures = Path(__file__).parent / "terraform_hours"
    with tempfile.TemporaryDirectory(prefix="expense-hours-terraform-") as temporary:
        work = Path(temporary)
        shutil.copy(root / "operating_hours.tf", work / "operating_hours.tf")
        shutil.copy(root / ".terraform.lock.hcl", work / ".terraform.lock.hcl")
        shutil.copy(fixtures / "stub.tf", work / "stub.tf")
        (work / "tests").mkdir()
        shutil.copy(fixtures / "hours.tftest.hcl", work / "tests/hours.tftest.hcl")
        (work / ".terraform").mkdir()
        (work / ".terraform/providers").symlink_to(root / ".terraform/providers", target_is_directory=True)
        for arguments in (("init", "-backend=false", "-input=false"), ("test", "-no-color")):
            subprocess.run(["terraform", f"-chdir={work}", *arguments], check=True)


if __name__ == "__main__":
    main()
