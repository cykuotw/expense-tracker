export interface OperatingHours {
    enabled: boolean;
    start: string;
    stop: string;
    safety_stop: string;
    timezone: string;
}

export function parseOperatingHours(value: unknown): OperatingHours | null {
    if (typeof value !== "object" || value === null) return null;
    const hours = value as OperatingHours;
    const clock = /^(?:[01]\d|2[0-3]):[0-5]\d$/;
    if (typeof hours.enabled !== "boolean" || !clock.test(hours.start) || !clock.test(hours.stop) || !clock.test(hours.safety_stop)) return null;
    try {
        new Intl.DateTimeFormat("en", { timeZone: hours.timezone });
    } catch { return null; }
    return hours;
}

export function outsideOperatingHours(hours: OperatingHours, now = new Date()) {
    const parts = new Intl.DateTimeFormat("en-GB", {
        timeZone: hours.timezone, hour: "2-digit", minute: "2-digit", hourCycle: "h23",
    }).formatToParts(now);
    const minute = (time: string) => Number(time.slice(0, 2)) * 60 + Number(time.slice(3));
    const current = Number(parts.find((p) => p.type === "hour")?.value) * 60 + Number(parts.find((p) => p.type === "minute")?.value);
    return (current - minute(hours.start) + 1440) % 1440 >= (minute(hours.stop) - minute(hours.start) + 1440) % 1440;
}
