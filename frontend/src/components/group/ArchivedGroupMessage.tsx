import { Link } from "react-router-dom";
import DesktopBackLink from "../DesktopBackLink";
import ArchivedStatus from "./ArchivedStatus";

export default function ArchivedGroupMessage({ groupId }: { groupId: string }) {
    return <main className="page-shell"><div className="page-container max-w-4xl">
        <DesktopBackLink to={`/group/${groupId}`} label="Back to group" />
        <div className="mb-4"><ArchivedStatus /></div>
        <h1 className="page-title">This group is archived</h1>
        <p className="page-copy">Its history is available to view. The group creator must restore it before anyone can make changes.</p>
        <Link className="ui-button ui-button-outline mt-6" to={`/group/${groupId}`}>View group</Link>
    </div></main>;
}
