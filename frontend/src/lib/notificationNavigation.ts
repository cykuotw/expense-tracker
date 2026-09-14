export function notificationDestination(origin: string, value: unknown) {
    const fallback = new URL("/", origin);
    if (typeof value !== "string") return fallback.href;
    try {
        const destination = new URL(value, fallback);
        return destination.origin === fallback.origin ? destination.href : fallback.href;
    } catch {
        return fallback.href;
    }
}
