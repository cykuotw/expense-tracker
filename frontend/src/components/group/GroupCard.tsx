import { Link } from "react-router-dom";
import { GroupCardData } from "../../types/group";

export default function GroupCard(groupData: GroupCardData) {
    return (
        <>
            <div className="card card-border w-full md:max-w-80 bg-base-100 shadow-md m-3">
                <Link to={`/group/${groupData.id}`}>
                    <div className="card-body">
                        <div className="card-title">{groupData.groupName}</div>
                        <p className="break-all">{groupData.description}</p>
                    </div>
                </Link>
            </div>
        </>
    );
}
