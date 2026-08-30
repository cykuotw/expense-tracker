from __future__ import annotations

import ipaddress
import json
import os
import re
import stat
from dataclasses import dataclass
from pathlib import Path
from typing import Any


class ConfigError(ValueError):
    pass


def _object(value: Any, name: str) -> dict[str, Any]:
    if not isinstance(value, dict):
        raise ConfigError(f"{name} must be an object")
    return value


def _keys(value: dict[str, Any], name: str, required: set[str], optional: set[str] = frozenset()) -> None:
    missing = required - value.keys()
    unknown = value.keys() - required - optional
    if missing:
        raise ConfigError(f"{name} is missing: {', '.join(sorted(missing))}")
    if unknown:
        raise ConfigError(f"{name} has unknown keys: {', '.join(sorted(unknown))}")


def _string(value: Any, name: str, *, allow_empty: bool = False) -> str:
    if not isinstance(value, str) or (not allow_empty and not value.strip()):
        raise ConfigError(f"{name} must be a{' non-empty' if not allow_empty else ''} string")
    return value.strip() if not allow_empty else value


def _integer(value: Any, name: str, *, minimum: int = 1) -> int:
    if isinstance(value, bool) or not isinstance(value, int) or value < minimum:
        raise ConfigError(f"{name} must be an integer >= {minimum}")
    return value


def _hostname(value: Any, name: str) -> str:
    result = _string(value, name).lower().rstrip(".")
    if not re.fullmatch(r"[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?", result):
        raise ConfigError(f"{name} must be a lowercase DNS hostname")
    return result


def _secret(value: Any, name: str, *, minimum: int = 16) -> str:
    result = _string(value, name)
    if len(result) < minimum or result.upper().startswith(("REPLACE", "CHANGE", "EXAMPLE")):
        raise ConfigError(f"{name} must be a non-placeholder secret of at least {minimum} characters")
    return result


@dataclass(frozen=True)
class Deployment:
    account_id: str
    name_prefix: str
    environment: str
    tags: dict[str, str]


@dataclass(frozen=True)
class AWS:
    region: str
    vpc_id: str
    subnet_id: str
    key_pair_name: str
    operator_ssh_cidr: str
    hosted_zone_name: str


@dataclass(frozen=True)
class Database:
    name: str
    admin_user: str
    admin_password: str
    migration_user: str
    migration_password: str
    runtime_user: str
    runtime_password: str
    instance_type: str
    ami_id: str | None


@dataclass(frozen=True)
class Backend:
    api_hostname: str
    google_client_id: str
    jwt_secret: str
    jwt_exp: int
    refresh_jwt_secret: str
    refresh_jwt_exp: int
    expenses_per_page: int
    db_conn_max_lifetime_seconds: int
    db_conn_max_idle_time_seconds: int


@dataclass(frozen=True)
class Frontend:
    hostname: str


@dataclass(frozen=True)
class FirstAdmin:
    email: str
    password: str
    firstname: str
    lastname: str
    nickname: str


@dataclass(frozen=True)
class LocalCredentials:
    ssh_private_key_file: Path
    google_id_token_file: Path | None


@dataclass(frozen=True)
class Config:
    deployment: Deployment
    aws: AWS
    database: Database
    backend: Backend
    frontend: Frontend
    first_admin: FirstAdmin | None
    local_credentials: LocalCredentials

    @property
    def frontend_origin(self) -> str:
        return f"https://{self.frontend.hostname}"

    @property
    def api_origin(self) -> str:
        return f"https://{self.backend.api_hostname}"

    def terraform_variables(self, *, temporary_access: bool) -> dict[str, Any]:
        return {
            "aws_region": self.aws.region,
            "expected_account_id": self.deployment.account_id,
            "name_prefix": self.deployment.name_prefix,
            "environment": self.deployment.environment,
            "tags": self.deployment.tags,
            "vpc_id": self.aws.vpc_id,
            "subnet_id": self.aws.subnet_id,
            "key_pair_name": self.aws.key_pair_name,
            "operator_ssh_cidr": self.aws.operator_ssh_cidr,
            "enable_temporary_public_access": temporary_access,
            "hosted_zone_name": self.aws.hosted_zone_name,
            "database_instance_type": self.database.instance_type,
            "database_ami_id": self.database.ami_id,
            "worker_artifact_path": str((Path(__file__).parent / "build/worker.zip").resolve()),
            "bootstrap_artifact_path": str((Path(__file__).parent / "build/bootstrap.zip").resolve()),
            "api_hostname": self.backend.api_hostname,
            "frontend_hostname": self.frontend.hostname,
            "google_client_id": self.backend.google_client_id,
        }

    def worker_environment(self, db_host: str) -> dict[str, dict[str, str]]:
        return {"Variables": {
            "MODE": "release", "API_URL": "/api/v0",
            "FRONTEND_ORIGIN": self.frontend_origin,
            "CORS_ALLOWED_ORIGINS": self.frontend_origin,
            "CORS_ALLOW_CREDENTIALS": "true", "AUTH_COOKIE_DOMAIN": "",
            "AUTH_COOKIE_SECURE": "true", "AUTH_COOKIE_SAME_SITE": "lax",
            "DB_PUBLIC_HOST": db_host, "DB_PORT": "5432",
            "DB_USER": self.database.runtime_user,
            "DB_PASSWORD": self.database.runtime_password,
            "DB_NAME": self.database.name, "DB_SSLMODE": "disable",
            "DB_MAX_OPEN_CONNS": "2", "DB_MAX_IDLE_CONNS": "1",
            "DB_CONN_MAX_LIFETIME_SECONDS": str(self.backend.db_conn_max_lifetime_seconds),
            "DB_CONN_MAX_IDLE_TIME_SECONDS": str(self.backend.db_conn_max_idle_time_seconds),
            "JWT_SECRET": self.backend.jwt_secret, "JWT_EXP": str(self.backend.jwt_exp),
            "REFRESH_JWT_SECRET": self.backend.refresh_jwt_secret,
            "REFRESH_JWT_EXP": str(self.backend.refresh_jwt_exp),
            "EXPENSES_PER_PAGE": str(self.backend.expenses_per_page),
            "GOOGLE_OAUTH_ENABLED": "true",
            "GOOGLE_CLIENT_ID": self.backend.google_client_id,
            "GOOGLE_EXCHANGE_MODE": "upstream_verified",
        }}

    def bootstrap_environment(self, db_host: str) -> dict[str, dict[str, str]]:
        variables = {
            "DB_PUBLIC_HOST": db_host, "DB_PORT": "5432", "DB_SSLMODE": "disable",
            "DB_MAINTENANCE_NAME": "postgres", "DB_NAME": self.database.name,
            "DB_ADMIN_USER": self.database.admin_user,
            "DB_ADMIN_PASSWORD": self.database.admin_password,
            "DB_MIGRATION_USER": self.database.migration_user,
            "DB_MIGRATION_PASSWORD": self.database.migration_password,
            "DB_RUNTIME_USER": self.database.runtime_user,
            "DB_RUNTIME_PASSWORD": self.database.runtime_password,
        }
        if self.first_admin:
            variables.update({
                "FIRST_ADMIN_EMAIL": self.first_admin.email,
                "FIRST_ADMIN_PASSWORD": self.first_admin.password,
                "FIRST_ADMIN_FIRSTNAME": self.first_admin.firstname,
                "FIRST_ADMIN_LASTNAME": self.first_admin.lastname,
                "FIRST_ADMIN_NICKNAME": self.first_admin.nickname,
            })
        return {"Variables": variables}


def _protected_path(value: Any, name: str, repo_root: Path, *, required: bool) -> Path | None:
    if value is None and not required:
        return None
    path = Path(_string(value, name))
    if not path.is_absolute():
        raise ConfigError(f"{name} must be an absolute path")
    if path.is_symlink() or not path.is_file():
        raise ConfigError(f"{name} must be an existing regular file, not a symlink")
    resolved = path.resolve()
    if resolved == repo_root or repo_root in resolved.parents:
        raise ConfigError(f"{name} must be outside the repository")
    if stat.S_IMODE(resolved.stat().st_mode) != 0o600:
        raise ConfigError(f"{name} must have mode 0600")
    return resolved


def load(path: Path, repo_root: Path) -> Config:
    if not path.is_absolute():
        raise ConfigError("DEPLOY_CONFIG_FILE must be absolute")
    if path.is_symlink() or not path.is_file():
        raise ConfigError("deployment config must be an existing regular file, not a symlink")
    resolved = path.resolve()
    if resolved == repo_root or repo_root in resolved.parents:
        raise ConfigError("deployment config must be outside the repository")
    if stat.S_IMODE(resolved.stat().st_mode) != 0o600:
        raise ConfigError("deployment config must have mode 0600")
    try:
        raw = _object(json.loads(resolved.read_text()), "config")
    except (OSError, json.JSONDecodeError) as exc:
        raise ConfigError(f"cannot read deployment config: {exc}") from exc
    required_sections = {"deployment", "aws", "database", "backend", "frontend", "local_credentials"}
    _keys(raw, "config", required_sections, {"first_admin"})

    deployment = _object(raw["deployment"], "deployment")
    _keys(deployment, "deployment", {"account_id", "name_prefix", "environment"}, {"tags"})
    account_id = _string(deployment["account_id"], "deployment.account_id")
    if not re.fullmatch(r"\d{12}", account_id):
        raise ConfigError("deployment.account_id must be 12 digits")
    tags_raw = _object(deployment.get("tags", {}), "deployment.tags")
    if not all(isinstance(key, str) for key in tags_raw):
        raise ConfigError("deployment.tags keys must be strings")
    tags = {key: _string(value, f"deployment.tags.{key}") for key, value in tags_raw.items()}
    name_prefix = _string(deployment["name_prefix"], "deployment.name_prefix")
    environment = _string(deployment["environment"], "deployment.environment")
    for value, name in ((name_prefix, "deployment.name_prefix"), (environment, "deployment.environment")):
        if not re.fullmatch(r"[a-z0-9][a-z0-9-]{1,31}", value):
            raise ConfigError(f"{name} must be a lowercase DNS-safe name")
    if len(f"{name_prefix}-{environment}") > 39:
        raise ConfigError("combined deployment name_prefix/environment is too long for the frontend bucket")
    deploy = Deployment(account_id, name_prefix, environment, tags)

    aws_raw = _object(raw["aws"], "aws")
    _keys(aws_raw, "aws", {"region", "vpc_id", "subnet_id", "key_pair_name", "operator_ssh_cidr", "hosted_zone_name"})
    cidr = _string(aws_raw["operator_ssh_cidr"], "aws.operator_ssh_cidr")
    try:
        network = ipaddress.ip_network(cidr, strict=False)
    except ValueError as exc:
        raise ConfigError("aws.operator_ssh_cidr must be a restricted IPv4 CIDR") from exc
    if network.version != 4 or network.prefixlen == 0:
        raise ConfigError("aws.operator_ssh_cidr must be a restricted IPv4 CIDR")
    aws = AWS(_string(aws_raw["region"], "aws.region"), _string(aws_raw["vpc_id"], "aws.vpc_id"), _string(aws_raw["subnet_id"], "aws.subnet_id"), _string(aws_raw["key_pair_name"], "aws.key_pair_name"), cidr, _hostname(aws_raw["hosted_zone_name"], "aws.hosted_zone_name"))

    db_raw = _object(raw["database"], "database")
    db_required = {"name", "admin_user", "admin_password", "migration_user", "migration_password", "runtime_user", "runtime_password", "instance_type"}
    _keys(db_raw, "database", db_required, {"ami_id"})
    users = [_string(db_raw[k], f"database.{k}") for k in ("admin_user", "migration_user", "runtime_user")]
    if len(set(users)) != 3:
        raise ConfigError("database users must be distinct")
    database = Database(_string(db_raw["name"], "database.name"), users[0], _secret(db_raw["admin_password"], "database.admin_password"), users[1], _secret(db_raw["migration_password"], "database.migration_password"), users[2], _secret(db_raw["runtime_password"], "database.runtime_password"), _string(db_raw["instance_type"], "database.instance_type"), None if db_raw.get("ami_id") is None else _string(db_raw["ami_id"], "database.ami_id"))
    if database.name == "postgres":
        raise ConfigError("database.name must differ from postgres")

    backend_raw = _object(raw["backend"], "backend")
    backend_required = {"api_hostname", "google_client_id", "jwt_secret", "jwt_exp", "refresh_jwt_secret", "refresh_jwt_exp", "expenses_per_page", "db_conn_max_lifetime_seconds", "db_conn_max_idle_time_seconds"}
    _keys(backend_raw, "backend", backend_required)
    google_client_id = _string(backend_raw["google_client_id"], "backend.google_client_id")
    if google_client_id.upper().startswith(("REPLACE", "EXAMPLE")):
        raise ConfigError("backend.google_client_id must not be a placeholder")
    backend = Backend(_hostname(backend_raw["api_hostname"], "backend.api_hostname"), google_client_id, _secret(backend_raw["jwt_secret"], "backend.jwt_secret", minimum=32), _integer(backend_raw["jwt_exp"], "backend.jwt_exp"), _secret(backend_raw["refresh_jwt_secret"], "backend.refresh_jwt_secret", minimum=32), _integer(backend_raw["refresh_jwt_exp"], "backend.refresh_jwt_exp"), _integer(backend_raw["expenses_per_page"], "backend.expenses_per_page"), _integer(backend_raw["db_conn_max_lifetime_seconds"], "backend.db_conn_max_lifetime_seconds"), _integer(backend_raw["db_conn_max_idle_time_seconds"], "backend.db_conn_max_idle_time_seconds"))

    frontend_raw = _object(raw["frontend"], "frontend")
    _keys(frontend_raw, "frontend", {"hostname"})
    frontend = Frontend(_hostname(frontend_raw["hostname"], "frontend.hostname"))
    for name, host in (("backend.api_hostname", backend.api_hostname), ("frontend.hostname", frontend.hostname)):
        if host != aws.hosted_zone_name and not host.endswith(f".{aws.hosted_zone_name}"):
            raise ConfigError(f"{name} must belong to aws.hosted_zone_name")

    admin = None
    if raw.get("first_admin") is not None:
        admin_raw = _object(raw["first_admin"], "first_admin")
        _keys(admin_raw, "first_admin", {"email", "password", "firstname", "lastname"}, {"nickname"})
        admin = FirstAdmin(_string(admin_raw["email"], "first_admin.email"), _secret(admin_raw["password"], "first_admin.password"), _string(admin_raw["firstname"], "first_admin.firstname"), _string(admin_raw["lastname"], "first_admin.lastname"), _string(admin_raw.get("nickname", ""), "first_admin.nickname", allow_empty=True))

    credentials_raw = _object(raw["local_credentials"], "local_credentials")
    _keys(credentials_raw, "local_credentials", {"ssh_private_key_file"}, {"google_id_token_file"})
    credentials = LocalCredentials(_protected_path(credentials_raw["ssh_private_key_file"], "local_credentials.ssh_private_key_file", repo_root, required=True), _protected_path(credentials_raw.get("google_id_token_file"), "local_credentials.google_id_token_file", repo_root, required=False))
    return Config(deploy, aws, database, backend, frontend, admin, credentials)


def template() -> dict[str, Any]:
    return {
        "deployment": {"account_id": "000000000000", "name_prefix": "expense-tracker", "environment": "serverless", "tags": {}},
        "aws": {"region": "ca-central-1", "vpc_id": "vpc-REPLACE", "subnet_id": "subnet-REPLACE", "key_pair_name": "REPLACE", "operator_ssh_cidr": "203.0.113.10/32", "hosted_zone_name": "example.com"},
        "database": {"name": "expense_tracker", "admin_user": "expense_admin", "admin_password": "REPLACE_WITH_RANDOM_SECRET", "migration_user": "expense_migration", "migration_password": "REPLACE_WITH_RANDOM_SECRET", "runtime_user": "expense_runtime", "runtime_password": "REPLACE_WITH_RANDOM_SECRET", "instance_type": "t4g.micro", "ami_id": None},
        "backend": {"api_hostname": "api.example.com", "google_client_id": "REPLACE.apps.googleusercontent.com", "jwt_secret": "REPLACE_WITH_32_BYTE_RANDOM_SECRET", "jwt_exp": 900, "refresh_jwt_secret": "REPLACE_WITH_32_BYTE_RANDOM_SECRET", "refresh_jwt_exp": 604800, "expenses_per_page": 25, "db_conn_max_lifetime_seconds": 300, "db_conn_max_idle_time_seconds": 60},
        "frontend": {"hostname": "expense.example.com"},
        "first_admin": None,
        "local_credentials": {"ssh_private_key_file": "/absolute/path/to/key.pem", "google_id_token_file": None},
    }


def initialize(path: Path, repo_root: Path) -> None:
    if not path.is_absolute():
        raise ConfigError("DEPLOY_CONFIG_FILE must be absolute")
    resolved_parent = path.parent.resolve()
    if resolved_parent == repo_root or repo_root in resolved_parent.parents:
        raise ConfigError("deployment config must be outside the repository")
    if path.exists():
        raise ConfigError(f"refusing to overwrite existing config: {path}")
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(descriptor, "w") as stream:
        json.dump(template(), stream, indent=2)
        stream.write("\n")
