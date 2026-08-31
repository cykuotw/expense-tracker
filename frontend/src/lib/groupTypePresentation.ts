import { mdiAccountGroup, mdiAccountMultiple, mdiAirplane, mdiCalendarStar, mdiHomeCityOutline, mdiTagOutline } from "@mdi/js";

export type GroupType = "trip" | "home" | "family" | "friends" | "event" | "other";

export const groupTypeOptions = [
    { value: "trip", label: "Trip", icon: mdiAirplane, iconClassName: "bg-rose-100 text-rose-700" },
    { value: "home", label: "Home", icon: mdiHomeCityOutline, iconClassName: "bg-amber-100 text-amber-700" },
    { value: "family", label: "Family", icon: mdiAccountGroup, iconClassName: "bg-teal-100 text-teal-700" },
    { value: "friends", label: "Friends", icon: mdiAccountMultiple, iconClassName: "bg-violet-100 text-violet-700" },
    { value: "event", label: "Event", icon: mdiCalendarStar, iconClassName: "bg-blue-100 text-blue-700" },
    { value: "other", label: "Other", icon: mdiTagOutline, iconClassName: "bg-slate-100 text-slate-700" },
] as const;

export function getGroupTypePresentation(value?: string) {
    return groupTypeOptions.find((option) => option.value === value) ?? groupTypeOptions[1];
}
