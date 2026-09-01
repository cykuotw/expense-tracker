import { useEffect, useState } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import { mdiCheckBold } from "@mdi/js";
import Icon from "@mdi/react";
import { GroupTypePicker } from "../components/group/GroupTypePicker";
import { GroupMemberManager } from "../components/group/GroupMemberManager";
import MobilePageHeader from "../components/MobilePageHeader";
import { AddMemberProvider } from "../contexts/AddMemberContext";
import { apiFetch, getResponseErrorMessage } from "../lib/api";

export default function EditGroup() {
    const { id } = useParams();
    const { hash } = useLocation();
    const navigate = useNavigate();
    const [form, setForm] = useState({
        groupName: "",
        description: "",
        currency: "CAD",
        groupType: "home",
    });
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        if (!id) return;
        void apiFetch(`/group/${id}`).then(async (response) => {
            if (!response.ok) return;
            const data = await response.json();
            setForm({
                groupName: data.groupName ?? "",
                description: data.description ?? "",
                currency: data.currency ?? "CAD",
                groupType: data.groupType ?? "home",
            });
        });
    }, [id]);

    useEffect(() => {
        if (hash !== "#members") return;

        document.getElementById("members")?.scrollIntoView({ block: "start" });
    }, [hash]);

    const save = async (event: React.FormEvent) => {
        event.preventDefault();
        if (!id || !form.groupName.trim()) return;
        setSaving(true);
        try {
            const response = await apiFetch(`/group/${id}`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    ...form,
                    groupName: form.groupName.trim(),
                    description: form.description.trim(),
                }),
            });
            if (!response.ok) {
                toast.error(
                    await getResponseErrorMessage(response, "Failed to update group"),
                );
                return;
            }
            toast.success("Group updated");
            navigate(`/group/${id}`);
        } finally {
            setSaving(false);
        }
    };

    return (
        <div className="page-shell compact-mobile-page">
            <div className="page-container max-w-4xl">
                <MobilePageHeader
                    title="Edit group"
                    backTo={`/group/${id}`}
                    backLabel="Back to group"
                    action={
                        saving ? (
                            <span className="ui-spinner ui-spinner-sm" role="status" aria-label="Saving group" />
                        ) : (
                            <button
                                type="submit"
                                form="edit-group-form"
                                className="ui-button ui-button-primary min-h-12 min-w-12 px-3"
                                aria-label="Save group"
                                disabled={!form.groupName.trim()}
                            >
                                <Icon path={mdiCheckBold} size={1} aria-hidden="true" />
                            </button>
                        )
                    }
                />
                <div className="page-header desktop-page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Group</div>
                        <h1 className="page-title">Edit group</h1>
                    </div>
                </div>
                <form
                    id="edit-group-form"
                    className="panel-card grid gap-3 rounded-[2rem] p-4 md:gap-5 md:p-8"
                    onSubmit={save}
                >
                    <Field
                        label="Group name"
                        value={form.groupName}
                        onChange={(groupName) =>
                            setForm((current) => ({ ...current, groupName }))
                        }
                        required
                    />
                    <Field
                        label="Description"
                        value={form.description}
                        onChange={(description) =>
                            setForm((current) => ({ ...current, description }))
                        }
                    />
                    <div>
                        <div className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                            Group type
                        </div>
                        <div className="mt-2">
                            <GroupTypePicker
                                value={form.groupType}
                                onChange={(groupType) =>
                                    setForm((current) => ({ ...current, groupType }))
                                }
                            />
                        </div>
                    </div>
                    <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
                        Currency
                        <select
                            className="ui-select mt-2 w-full"
                            value={form.currency}
                            onChange={(event) =>
                                setForm((current) => ({
                                    ...current,
                                    currency: event.target.value,
                                }))
                            }
                        >
                            <option>CAD</option>
                            <option>USD</option>
                            <option>NTD</option>
                        </select>
                    </label>
                    <div className="hidden flex-col gap-3 md:flex md:flex-row md:justify-between">
                        <button
                            className="ui-button ui-button-primary"
                            disabled={saving || !form.groupName.trim()}
                        >
                            {saving ? "Saving…" : "Save group"}
                        </button>
                        <Link className="ui-button ui-button-ghost" to={`/group/${id}`}>
                            Cancel
                        </Link>
                    </div>
                </form>
                <section id="members" className="mt-4 scroll-mt-4 md:mt-6 md:scroll-mt-6">
                    <div className="mb-3 md:mb-4">
                        <div className="page-eyebrow">Members</div>
                        <h2 className="mt-1 text-xl font-semibold text-foreground md:text-2xl">
                            Manage members
                        </h2>
                        <p className="mt-1 text-sm text-foreground/65">
                            Select the people who belong to this group, then
                            update members separately from the group details.
                        </p>
                    </div>
                    <AddMemberProvider
                        groupId={id ?? null}
                        returnToGroupAfterSave={false}
                    >
                        <GroupMemberManager />
                    </AddMemberProvider>
                </section>
            </div>
        </div>
    );
}

function Field({
    label,
    value,
    onChange,
    required = false,
}: {
    label: string;
    value: string;
    onChange: (value: string) => void;
    required?: boolean;
}) {
    return (
        <label className="text-xs font-semibold uppercase tracking-[0.2em] text-foreground/60">
            {label}
            <span className="ui-input-shell mt-2 flex w-full bg-background">
                <input
                    className="grow"
                    value={value}
                    required={required}
                    onChange={(event) => onChange(event.target.value)}
                />
            </span>
        </label>
    );
}
