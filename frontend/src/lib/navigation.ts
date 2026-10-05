export type NavigationKind = "POP" | "PUSH" | "REPLACE";
export type ViewValue = string | number | boolean;
export interface NavigationEntry {
    key: string;
    url: string;
    parent?: string;
    view: Map<string, ViewValue>;
    position?: { top: number; left: number; href?: string; id?: string };
}

const ID = "[A-Za-z0-9_-]+";
const routes = [
    /^\/$/, /^\/account$/, /^\/admin\/users$/, /^\/create_group$/,
    /^\/add_member$/, /^\/create_expense(?:\/split|\/receipt(?:\/review)?)?$/,
    new RegExp(`^/group/${ID}(?:/edit|/monthly-review/\\d{4}-(?:0[1-9]|1[0-2]))?$`),
    new RegExp(`^/expense/${ID}(?:/edit(?:/split)?)?$`),
];

export function internalDestination(value: string): string | undefined {
    if (value.length > 2048 || !value.startsWith("/") || value.startsWith("//") || /[\\\s%]/.test(value.split(/[?#]/)[0])) return;
    let url: URL;
    try { url = new URL(value, "https://navigation.invalid"); } catch { return; }
    if (url.origin !== "https://navigation.invalid" || !routes.some((route) => route.test(url.pathname))) return;
    if (url.pathname.split("/").some((part) => part === "null" || part === "undefined")) return;
    return `${url.pathname}${url.search}${url.hash}`;
}

function pathname(url: string) { return url.split(/[?#]/)[0]; }
function reviewGroup(url: string) { return pathname(url).match(/^\/group\/([^/]+)\/monthly-review\//)?.[1]; }
function draftRoot(url: string) {
    const path = pathname(url);
    if (path === "/create_expense" || path.startsWith("/create_expense/")) return "/create_expense";
    return path.match(/^\/expense\/[^/]+\/edit(?:\/|$)/)?.[0].replace(/\/$/, "");
}

export function destinationLabel(url: string) {
    const path = pathname(url);
    if (reviewGroup(url)) return "Back to monthly review";
    if (/^\/group\/[^/]+$/.test(path)) return "Back to group";
    if (/^\/expense\/[^/]+$/.test(path)) return "Back to expense";
    if (path === "/account") return "Back to settings";
    if (path === "/create_expense" || /^\/expense\/[^/]+\/edit$/.test(path)) return "Back to expense form";
    return "Back to groups";
}

// Route-specific defaults live here; owning-group information comes from loaded data.
export function routeFallback(url: string, supplied: string) {
    const path = pathname(url);
    const group = path.match(/^\/group\/([^/]+)\//)?.[1];
    const expense = path.match(/^\/expense\/([^/]+)\/edit$/)?.[1];
    if (group) return internalDestination(`/group/${group}`) ?? "/";
    if (expense) return internalDestination(`/expense/${expense}`) ?? "/";
    if (path === "/admin/users") return "/account";
    return internalDestination(supplied) ?? "/";
}

function allowedSource(current: string, source: string) {
    const path = pathname(current), from = pathname(source);
    if (path === from || !internalDestination(source)) return false;
    const expense = path.match(/^\/expense\/([^/]+)\/edit$/)?.[1];
    if (expense) return from === `/expense/${expense}`;
    if (path.endsWith("/split")) return from === path.slice(0, -6);
    if (path.startsWith("/create_expense/receipt")) return from === "/create_expense";
    if (/^\/group\/[^/]+$/.test(path)) return from === "/";
    if (path === "/create_group") return from === "/";
    if (/^\/group\/[^/]+\/edit$/.test(path)) return from === path.slice(0, -5);
    if (path === "/add_member") return /^\/group\/[^/]+$/.test(from);
    if (path === "/create_expense" || /^\/expense\/[^/]+$/.test(path)) return from === "/" || /^\/group\/[^/]+$/.test(from) || !!reviewGroup(source);
    if (reviewGroup(current)) return from === "/" || from === `/group/${reviewGroup(current)}`;
    if (path === "/admin/users") return from === "/" || from === "/account" || /^\/group\/[^/]+$/.test(from) || !!reviewGroup(source) || /^\/expense\/[^/]+$/.test(from);
    return false;
}

/** Bounded, tab-local provenance. Never serializes API responses or form drafts. */
export class NavigationStore {
    entries = new Map<string, NavigationEntry>();
    history: string[] = [];
    index = -1;
    blockers = new Map<string, Set<symbol>>();
    private pending?: NavigationEntry;

    observe(key: string, url: string, kind: NavigationKind) {
        if (this.history[this.index] === key) return false;
        const previous = this.entries.get(this.history[this.index]);
        const knownIndex = this.history.indexOf(key);
        if (kind === "POP" && knownIndex >= 0) {
            this.index = knownIndex;
            return true;
        }
        const parent = kind !== "POP" && previous && allowedSource(url, previous.url) ? previous.key : undefined;
        const sameReview = kind !== "POP" && previous && reviewGroup(url) && reviewGroup(url) === reviewGroup(previous.url);
        const root = draftRoot(url);
        const sameDraftStep = kind !== "POP" && previous && root && root === draftRoot(previous.url) && pathname(url) !== root && pathname(previous.url) !== root;
        const entry: NavigationEntry = {
            key, url, parent: sameReview || sameDraftStep ? previous!.parent : parent,
            view: new Map(),
        };
        // A fallback replacement can retain the destination's own origin/view.
        if (this.pending?.url === url) {
            entry.parent = this.pending.parent;
            entry.view = new Map(this.pending.view);
            entry.position = this.pending.position;
        } else if (kind === "REPLACE" && previous && !parent) {
            const inherited = previous.parent && this.entries.get(previous.parent);
            if (inherited && allowedSource(url, inherited.url)) entry.parent = inherited.key;
        }
        this.pending = undefined;
        this.entries.set(key, entry);
        if (kind === "REPLACE" && this.index >= 0) this.history[this.index] = key;
        else if (kind === "POP") { this.history = [key]; this.index = 0; }
        else { this.history = this.history.slice(0, this.index + 1); this.history.push(key); this.index++; }
        while (this.entries.size > 80) {
            const oldest = this.entries.keys().next().value!;
            this.entries.delete(oldest);
            this.blockers.delete(oldest);
        }
        if (this.history.length > 80) {
            const removed = this.history.length - 80;
            this.history.splice(0, removed);
            this.index -= removed;
        }
        return true;
    }

    target(key: string, url: string, fallback: string) {
        const entry = this.entries.get(key);
        const source = entry?.parent ? this.entries.get(entry.parent) : undefined;
        if (source && allowedSource(url, source.url)) return source;
        return { url: routeFallback(url, fallback) };
    }

    returnAction(key: string, url: string, fallback: string, explicit?: string) {
        const target = this.target(key, url, fallback);
        const destination = explicit ? internalDestination(explicit) ?? "/" : target.url;
        // Search only verified predecessors, never history.length or router internals.
        let targetIndex = -1;
        for (let index = this.index - 1; index >= 0; index--) {
            const entry = this.entries.get(this.history[index]);
            if (entry?.url === destination && (explicit || ("key" in target && entry.key === target.key))) { targetIndex = index; break; }
        }
        if (this.history[this.index] === key && targetIndex >= 0) return { delta: targetIndex - this.index, url: destination };
        const source = "key" in target && target.url === destination ? target : undefined;
        this.pending = source;
        return { url: destination };
    }
}
