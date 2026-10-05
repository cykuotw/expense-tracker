import { useRef, useState } from "react";
import Icon from "@mdi/react";
import { mdiArchiveArrowUpOutline, mdiArchiveOutline } from "@mdi/js";
import { cn } from "../../lib/utils";
import ArchivedStatus from "./ArchivedStatus";
import { toast } from "react-hot-toast";
import { apiFetch, getResponseErrorMessage } from "../../lib/api";
import { useReturnNavigation } from "../../hooks/navigation";
import { usePWAUpdateBlocker } from "../../hooks/usePWAUpdateBlocker";
import type { GroupInfo } from "../../types/group";
import { Button } from "../ui/button";
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from "../ui/alert-dialog";

export default function GroupLifecycleControls({ groupId, group, onRefresh }: { groupId: string; group: GroupInfo; onRefresh: () => void }) {
    const [open, setOpen] = useState(false);
    const [pending, setPending] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const inFlight = useRef(false);
    const { finish } = useReturnNavigation("/");
    usePWAUpdateBlocker(open || pending);
    const archived = group.isActive === false;
    const changeStatus = async () => {
        if (inFlight.current || (archived ? !group.canRestore : !group.canArchive)) return;
        inFlight.current = true;
        setPending(true);
        setError(null);
        try {
            const response = await apiFetch(`/${archived ? "restore" : "archive"}_group/${encodeURIComponent(groupId)}`, { method: "PUT" });
            if (!response.ok) {
                setError(await getResponseErrorMessage(response, `Could not ${archived ? "restore" : "archive"} the group. Try again.`));
                onRefresh();
                return;
            }
            toast.success(archived ? "Group restored" : "Group archived");
            finish(archived ? `/group/${groupId}` : "/");
        } catch {
            setError("Could not update group status. Check your connection and try again.");
        } finally {
            inFlight.current = false;
            setPending(false);
            setOpen(false);
        }
    };
    if (!group.canManageLifecycle) return null;
    return <section id="status" className={cn("mt-6 scroll-mt-6 p-6 md:p-8", archived ? "rounded-2xl border border-border bg-muted shadow-sm" : "panel-card")} aria-labelledby="group-status-heading" aria-busy={pending}>
        <h2 id="group-status-heading" className="text-xl font-semibold">Group status</h2>
        {archived ? <div className="mt-3"><ArchivedStatus /></div> : null}
        <p className="mt-2 text-sm text-muted-foreground">{archived ? "Archived. History is retained; restore this group to make changes again." : "Archive a completed group to remove it from your active list. Its history stays available, and you can restore it later."}</p>
        {group.archiveBlockedReason ? <p id="archive-blocked-reason" className="mt-3 text-sm">{group.archiveBlockedReason}</p> : null}
        {error ? <p role="alert" className="mt-3 text-sm text-destructive">{error}</p> : null}
        {archived ? <Button className="mt-4 min-h-11" disabled={pending || !group.canRestore} onClick={() => void changeStatus()}><Icon path={mdiArchiveArrowUpOutline} data-icon="inline-start" aria-hidden="true" />{pending ? "Restoring…" : "Restore group"}</Button> :
            <AlertDialog open={open} onOpenChange={(next) => { if (!pending) setOpen(next); }}>
                <AlertDialogTrigger asChild><Button className="mt-4 min-h-11" variant="muted" disabled={pending || !group.canArchive} aria-describedby={group.archiveBlockedReason ? "archive-blocked-reason" : undefined}><Icon path={mdiArchiveOutline} data-icon="inline-start" aria-hidden="true" />Archive group</Button></AlertDialogTrigger>
                <AlertDialogContent>
                    <AlertDialogHeader><AlertDialogTitle>Archive this group?</AlertDialogTitle><AlertDialogDescription>The group will leave your active list. Expenses, balances, members, and history are retained. You can restore it from Archived groups later. Unsaved form changes are not saved.</AlertDialogDescription></AlertDialogHeader>
                    <AlertDialogFooter><AlertDialogCancel disabled={pending}>Cancel</AlertDialogCancel><AlertDialogAction disabled={pending} onClick={(event) => { event.preventDefault(); void changeStatus(); }}><Icon path={mdiArchiveOutline} data-icon="inline-start" aria-hidden="true" />{pending ? "Archiving…" : "Confirm archive"}</AlertDialogAction></AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>}
    </section>;
}
