interface GroupCardProps {
    Id: string;
    GroupName: string;
    Description: string;
}
export default function GroupCard({
    Id,
    GroupName,
    Description,
}: GroupCardProps) {
    return (
        <>
            <div className="card card-border w-full md:max-w-80 bg-base-100 shadow-md m-3">
                <a href={`/group/${Id}`}>
                    <div className="card-body">
                        <div className="card-title">{GroupName}</div>
                        <p className="break-all">{Description}</p>
                    </div>
                </a>
            </div>
        </>
    );
}
