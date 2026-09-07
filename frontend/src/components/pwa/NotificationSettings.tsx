import { useCallback, useEffect, useState } from "react";

import { apiFetch, asArray, getResponseErrorMessage } from "../../lib/api";
import { GroupListItem } from "../../types/group";

type Settings = {
    enabled: boolean;
    showDetails: boolean;
    vapidPublicKey: string;
    mutedGroups: Array<{ groupId: string; groupName: string; muted: boolean }>;
};

const EMPTY_SETTINGS: Settings = {
    enabled: false,
    showDetails: false,
    vapidPublicKey: "",
    mutedGroups: [],
};

function supportsPush() {
    return window.isSecureContext && "Notification" in window && "PushManager" in window && "serviceWorker" in navigator;
}

function base64URLToBytes(value: string) {
    const padded = value.replace(/-/g, "+").replace(/_/g, "/").padEnd(
        Math.ceil(value.length / 4) * 4,
        "=",
    );
    const decoded = window.atob(padded);
    return Uint8Array.from(decoded, (character) => character.charCodeAt(0));
}

export default function NotificationSettings() {
    const [settings, setSettings] = useState(EMPTY_SETTINGS);
    const [groups, setGroups] = useState<GroupListItem[]>([]);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState("");

    const load = useCallback(async () => {
        setLoading(true);
        setError("");
        try {
            const [settingsResponse, groupsResponse] = await Promise.all([
                apiFetch("/notifications/settings"),
                apiFetch("/groups"),
            ]);
            if (!settingsResponse.ok) {
                throw new Error(await getResponseErrorMessage(settingsResponse, "Unable to load notification settings"));
            }
            const settingsData = (await settingsResponse.json()) as Partial<Settings>;
            setSettings({
                enabled: settingsData.enabled === true,
                showDetails: settingsData.showDetails === true,
                vapidPublicKey: typeof settingsData.vapidPublicKey === "string"
                    ? settingsData.vapidPublicKey
                    : "",
                mutedGroups: Array.isArray(settingsData.mutedGroups)
                    ? settingsData.mutedGroups
                    : [],
            });
            if (groupsResponse.ok) {
                setGroups(asArray<GroupListItem>(await groupsResponse.json()));
            }
        } catch (reason) {
            setError(reason instanceof Error ? reason.message : "Unable to load notification settings");
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        if (!supportsPush()) {
            setLoading(false);
            return;
        }
        void load();
    }, [load]);

    const currentSubscription = async () => (await navigator.serviceWorker.ready).pushManager.getSubscription();
    const requestEnable = async () => {
        if (!supportsPush() || !settings.vapidPublicKey) return;
        setSaving(true); setError("");
        try {
            if (await Notification.requestPermission() !== "granted") {
                throw new Error("Notification permission was not granted. You can continue using the app without notifications.");
            }
            const registration = await navigator.serviceWorker.ready;
            const subscription = await registration.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: base64URLToBytes(settings.vapidPublicKey) });
            const json = subscription.toJSON();
            const response = await apiFetch("/notifications/subscriptions", {
                method: "POST",
                body: JSON.stringify({ endpoint: subscription.endpoint, p256dh: json.keys?.p256dh, auth: json.keys?.auth }),
            });
            if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Unable to enable notifications"));
            await load();
        } catch (reason) {
            setError(reason instanceof Error ? reason.message : "Unable to enable notifications");
        } finally { setSaving(false); }
    };
    const disable = async () => {
        setSaving(true); setError("");
        try {
            const subscription = await currentSubscription();
            if (subscription) {
                const response = await apiFetch("/notifications/subscriptions", { method: "DELETE", body: JSON.stringify({ endpoint: subscription.endpoint }) });
                if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Unable to disable notifications"));
                await subscription.unsubscribe();
            }
            await load();
        } catch (reason) { setError(reason instanceof Error ? reason.message : "Unable to disable notifications"); } finally { setSaving(false); }
    };
    const setDetails = async (showDetails: boolean) => {
        const subscription = await currentSubscription();
        if (!subscription) return;
        setSaving(true); setError("");
        try {
            const response = await apiFetch("/notifications/subscriptions/details", { method: "PATCH", body: JSON.stringify({ endpoint: subscription.endpoint, showDetails }) });
            if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Unable to update notification previews"));
            await load();
        } catch (reason) { setError(reason instanceof Error ? reason.message : "Unable to update notification previews"); } finally { setSaving(false); }
    };
    const setMute = async (groupID: string, muted: boolean) => {
        setSaving(true); setError("");
        try {
            const response = await apiFetch(`/notifications/groups/${groupID}/mute`, { method: "PUT", body: JSON.stringify({ muted }) });
            if (!response.ok) throw new Error(await getResponseErrorMessage(response, "Unable to update group notifications"));
            await load();
        } catch (reason) { setError(reason instanceof Error ? reason.message : "Unable to update group notifications"); } finally { setSaving(false); }
    };
    const muted = new Set(settings.mutedGroups.map((group) => group.groupId));
    const available = supportsPush() && Boolean(settings.vapidPublicKey);

    return <section className="panel-card rounded-[2rem] p-4 sm:p-7 lg:col-span-2" aria-labelledby="notifications-heading">
        <div className="section-label">This device</div><h2 id="notifications-heading" className="mt-2 text-xl font-semibold">Activity notifications</h2>
        <p className="mt-2 max-w-2xl text-sm leading-6 text-foreground/65">Notifications are off by default. When enabled, this device receives a generic alert when another member adds an expense. Details stay hidden unless you choose otherwise.</p>
        {!available ? <p className="mt-4 text-sm leading-6 text-foreground/65">This browser or deployment does not support web push. On iPhone and iPad, add Expense Tracker to the Home Screen from Safari first, then open the installed app to enable notifications.</p> : null}
        {error ? <p className="mt-4 text-sm text-destructive" role="alert">{error}</p> : null}
        {loading ? <div className="mt-5"><span className="ui-spinner ui-spinner-sm" aria-label="Loading notification settings" /></div> : <div className="mt-5 space-y-5">
            <button type="button" className="ui-button ui-button-primary min-h-11" disabled={!available || saving} onClick={() => void (settings.enabled ? disable() : requestEnable())}>{saving ? "Saving…" : settings.enabled ? "Disable notifications" : "Enable notifications"}</button>
            {settings.enabled ? <label className="flex items-start gap-3 text-sm leading-6"><input className="mt-1 size-5" type="checkbox" checked={settings.showDetails} disabled={saving} onChange={(event) => void setDetails(event.target.checked)} /><span><span className="font-medium text-foreground">Show notification details on this device</span><span className="block text-foreground/65">This may reveal the group name, currency, and amount on your lock screen. Disabling it hides future unsent notifications.</span></span></label> : null}
            {groups.length > 0 ? <fieldset disabled={saving}><legend className="text-sm font-medium text-foreground">Mute a group</legend><p className="mt-1 text-sm leading-6 text-foreground/65">Muted groups do not send activity notifications to this device.</p><div className="mt-3 grid gap-2 sm:grid-cols-2">{groups.map((group) => <label key={group.id} className="flex min-h-11 items-center gap-3 rounded-xl border border-border px-3 text-sm"><input className="size-5" type="checkbox" checked={!muted.has(group.id)} onChange={(event) => void setMute(group.id, !event.target.checked)} /><span>{group.groupName}</span></label>)}</div></fieldset> : null}
        </div>}
    </section>;
}
