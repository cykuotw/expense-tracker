"""Non-secret operating-window policy shared by configuration and operations."""
from __future__ import annotations

import re
from dataclasses import asdict, dataclass
from datetime import UTC, datetime, timedelta
from zoneinfo import ZoneInfo, ZoneInfoNotFoundError


def minute(value: str) -> int:
    if not isinstance(value, str) or not re.fullmatch(r"(?:[01]\d|2[0-3]):[0-5]\d", value):
        raise ValueError("operating hours must use 24-hour HH:MM format")
    hour, minutes = map(int, value.split(":"))
    return hour * 60 + minutes


def clock(value: int) -> str:
    return f"{value % 1440 // 60:02}:{value % 60:02}"


def operating_occurrences(hours: OperatingHours):
    """Scheduler's first occurrence of a repeated time, over known zone rules."""
    zone = ZoneInfo(hours.timezone)
    first = datetime.now(UTC).date().replace(month=1, day=1)
    for day in range(366 * 5):
        date = first + timedelta(days=day)
        def at(value: str, date=date) -> datetime:
            value_minute = minute(value)
            return datetime(date.year, date.month, date.day, value_minute // 60, value_minute % 60, tzinfo=zone)
        start = at(hours.start)
        stop = at(hours.stop) + timedelta(days=int(minute(hours.stop) <= minute(hours.start)))
        safety = at(hours.safety_stop, stop.date()) + timedelta(days=int(minute(hours.safety_stop) <= minute(hours.stop)))
        next_start = start + timedelta(days=1)
        yield start, stop, safety, next_start


@dataclass(frozen=True)
class OperatingHours:
    enabled: bool = False
    start: str = "09:00"
    stop: str = "22:00"
    safety_stop: str = "01:00"
    timezone: str = "America/Toronto"
    configured: bool = False

    @property
    def duration(self) -> int:
        return (minute(self.stop) - minute(self.start)) % 1440

    @property
    def review_time(self) -> str:
        return clock(minute(self.start) + 30)

    def contains(self, now: datetime) -> bool:
        local = now.astimezone(ZoneInfo(self.timezone))
        return (local.hour * 60 + local.minute - minute(self.start)) % 1440 < self.duration

    def public(self) -> dict:
        value = asdict(self)
        value.pop("configured")
        return value


def validate_wall_times(zone: str, times: list[str]) -> None:
    if not isinstance(zone, str) or (zone != "UTC" and "/" not in zone):
        raise ValueError("operating_hours.timezone must be an IANA zone, not EST or a fixed offset")
    try:
        timezone = ZoneInfo(zone)
    except (ZoneInfoNotFoundError, ValueError) as error:
        raise ValueError("operating_hours.timezone is not an installed IANA zone") from error
    minutes = [minute(value) for value in times]
    # Reject daily actions that would be skipped in a known spring-forward gap.
    # Revalidate on every apply; timezone database rules can change in the future.
    first = datetime.now(UTC).date().replace(month=1, day=1)
    for day in range(366 * 5):
        date = first + timedelta(days=day)
        for value in minutes:
            naive = datetime(date.year, date.month, date.day, value // 60, value % 60)
            local = naive.replace(tzinfo=timezone)
            if local.astimezone(UTC).astimezone(timezone).replace(tzinfo=None) != naive:
                raise ValueError(f"daily time {clock(value)} falls in a daylight-saving gap in {zone}; choose another time")


def parse(raw: object) -> OperatingHours:
    if raw is None:
        return OperatingHours()
    if not isinstance(raw, dict) or set(raw) - {"enabled", "start", "stop", "safety_stop", "timezone"}:
        raise ValueError("operating_hours must contain only enabled, start, stop, safety_stop, timezone")
    value = OperatingHours(**raw, configured=True)
    if not isinstance(value.enabled, bool):
        raise ValueError("operating_hours.enabled must be a boolean")
    validate_wall_times(value.timezone, [value.start, value.stop, value.safety_stop, value.review_time])
    safety_offset = (minute(value.safety_stop) - minute(value.stop)) % 1440
    offline = 1440 - value.duration
    if value.duration < 120 or offline < 120 or not 15 <= safety_offset <= offline - 15:
        raise ValueError("operating_hours requires at least two hours on/off and a safety stop at least 15 minutes inside the off period")
    for start, stop, safety, next_start in operating_occurrences(value):
        if (stop.astimezone(UTC) - start.astimezone(UTC)) < timedelta(hours=2) or (next_start.astimezone(UTC) - stop.astimezone(UTC)) < timedelta(hours=2):
            raise ValueError("operating_hours on/off periods must remain at least two elapsed hours across daylight-saving transitions")
        if min(safety.astimezone(UTC) - stop.astimezone(UTC), next_start.astimezone(UTC) - safety.astimezone(UTC)) < timedelta(minutes=15):
            raise ValueError("safety stop must remain 15 elapsed minutes inside the off period across daylight-saving transitions")
    return value


def validate_backup(hours: OperatingHours, time: str, timezone: str) -> None:
    if not hours.configured:
        return
    if timezone != hours.timezone:
        raise ValueError("backup.timezone must match operating_hours.timezone; remove backup.timezone to inherit the shared timezone")
    validate_wall_times(timezone, [time[:5]])
    offset = (minute(time[:5]) - minute(hours.start)) % 1440
    offset += int(time[6:]) / 60
    # Includes the existing 15-minute timer jitter and 45 minutes for dump/upload.
    if offset < 30 or offset > hours.duration - 60:
        raise ValueError("backup.time must be inside operating hours, at least 30 minutes after start and 60 minutes before stop")
    for start, stop, _, _ in operating_occurrences(hours):
        hour, minutes, seconds = map(int, time.split(":"))
        backup = start.replace(hour=hour, minute=minutes, second=seconds) + timedelta(days=int(minute(time[:5]) < minute(hours.start)))
        if backup.astimezone(UTC) - start.astimezone(UTC) < timedelta(minutes=30) or stop.astimezone(UTC) - backup.astimezone(UTC) < timedelta(minutes=60):
            raise ValueError("backup must retain 30/60 elapsed minute startup/shutdown margins across daylight-saving transitions")


def backup_freshness(time: str, timezone: str, latest: datetime | None, now: datetime) -> str:
    """Compare against the latest *due* daily backup, not a fixed 24h age."""
    local = now.astimezone(ZoneInfo(timezone))
    hour, minutes, seconds = map(int, time.split(":"))
    due = local.replace(hour=hour, minute=minutes, second=seconds, microsecond=0)
    # Allow 15m timer jitter plus 45m completion/upload, including DST days.
    if now < due.astimezone(UTC) + timedelta(hours=1):
        due -= timedelta(days=1)
    if latest is None:
        return "missing"
    return "fresh" if latest.astimezone(UTC) >= due.astimezone(UTC) else "stale"
