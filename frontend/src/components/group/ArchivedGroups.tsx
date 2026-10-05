import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import Icon from "@mdi/react";
import { mdiArchiveOutline, mdiChevronDown } from "@mdi/js";
import { cn } from "../../lib/utils";
import ArchivedStatus from "./ArchivedStatus";
import { useNavigationReady, useNavigationState } from "../../hooks/navigation";
import { apiFetch, asArray, getResponseErrorMessage } from "../../lib/api";
import type { GroupCardData } from "../../types/group";

export default function ArchivedGroups() {
    const [open, setOpen] = useNavigationState<boolean>("home.archived.open", false);
    const [depth, setDepth] = useNavigationState<number>("home.archived.pages", 1);
    const [groups, setGroups] = useState<GroupCardData[]>([]);
    const [pages, setPages] = useState(0);
    const [hasMore, setHasMore] = useState(true);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const state = useRef({ pages: 0, cursor: "", hasMore: true });
    const request = useRef<AbortController | null>(null);
    useNavigationReady(!open || Boolean(error) || (!loading && (pages >= depth || !hasMore)));

    const load = useCallback(async (target: number) => {
        if (request.current || !state.current.hasMore) return;
        const controller = new AbortController();
        request.current = controller;
        setLoading(true);
        setError(null);
        try {
            while (state.current.pages < target && state.current.hasMore) {
                const query = new URLSearchParams({ limit: "20" });
                if (state.current.cursor) query.set("cursor", state.current.cursor);
                const response = await apiFetch(`/groups/archived?${query}`, { signal: controller.signal });
                if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Could not load archived groups. Try again."));
                const data = await response.json() as { groups: GroupCardData[]; nextCursor?: string };
                if (controller.signal.aborted) return;
                const next = asArray<GroupCardData>(data.groups);
                setGroups((current) => Array.from(new Map([...current, ...next].map((group) => [group.id, group])).values()));
                state.current.pages++;
                state.current.cursor = data.nextCursor ?? "";
                state.current.hasMore = Boolean(state.current.cursor);
                setPages(state.current.pages);
                setHasMore(state.current.hasMore);
            }
            setDepth(state.current.pages);
        } catch (cause) {
            if (!controller.signal.aborted) setError(cause instanceof Error ? cause.message : "Could not load archived groups. Try again.");
        } finally {
            if (request.current === controller) request.current = null;
            if (!controller.signal.aborted) setLoading(false);
        }
    }, [setDepth]);
    useEffect(() => {
        if (open && state.current.pages === 0 && !error) void load(Math.max(1, depth));
    }, [open, depth, error, load]);
    useEffect(() => () => { request.current?.abort(); request.current = null; }, []);

    return <section className="mt-8 overflow-hidden rounded-xl border border-border bg-muted" aria-labelledby="archived-groups-heading">
        <h2 id="archived-groups-heading">
            <button type="button" className={cn("flex min-h-14 w-full items-center justify-between gap-4 px-4 py-3 text-left font-semibold text-muted-foreground transition-colors hover:bg-secondary hover:text-secondary-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary", open && "border-b border-border")} aria-expanded={open} aria-controls="archived-group-list" onClick={() => setOpen((value) => !value)}>
                <span className="flex items-center gap-2"><Icon path={mdiArchiveOutline} size={0.8} aria-hidden="true" />Archived groups</span>
                <Icon path={mdiChevronDown} size={0.9} className={cn("shrink-0 transition-transform motion-reduce:transition-none", open && "rotate-180")} aria-hidden="true" />
            </button>
        </h2>
        <div id="archived-group-list" hidden={!open} aria-busy={loading}>
        {open ? <div className="flex flex-col gap-4 p-4">
            <p className="text-sm text-muted-foreground">Completed groups and their history. Only the group creator can restore a group.</p>
            <ul className="flex flex-col gap-3">{groups.map((group) => <li key={group.id}><Link className="flex min-h-14 items-center justify-between gap-4 rounded-xl border border-border bg-muted px-4 py-3 text-muted-foreground shadow-sm transition-colors hover:bg-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary" to={`/group/${group.id}`} aria-label={`${group.groupName}, Archived`}><span className="min-w-0 break-words font-medium">{group.groupName}</span><ArchivedStatus /></Link></li>)}</ul>
            {loading ? <p role="status">Loading archived groups…</p> : null}
            {error ? <div role="alert"><p>{error}</p><button type="button" className="ui-button ui-button-outline mt-2" disabled={loading} onClick={() => void load(Math.max(depth, state.current.pages + 1))}>Try again</button></div> : null}
            {!loading && !error && pages > 0 && groups.length === 0 ? <p>No archived groups yet.</p> : null}
            {!loading && !error && pages > 0 && hasMore ? <button type="button" className="ui-button ui-button-outline self-start" onClick={() => void load(state.current.pages + 1)}>Load more archived groups</button> : null}
            {!loading && !error && groups.length > 0 && !hasMore ? <p className="text-sm text-muted-foreground">No more archived groups</p> : null}
        </div> : null}
        </div>
    </section>;
}
