import { Link } from "react-router-dom";
import GroupCard from "../components/group/GroupCard";
import { HomeProvider } from "../contexts/HomeContext";
import { useHome } from "../contexts/HomeContextHooks";

const HomeContent = () => {
    const { groupCards, loading } = useHome();

    if (loading) {
        return (
            <div className="flex justify-center items-center h-screen">
                <span className="loading loading-spinner loading-lg"></span>
            </div>
        );
    }

    const hasGroups = groupCards.length > 0;

    return (
        <div className="min-h-screen bg-gradient-to-br from-base-200 via-base-100 to-base-200 pb-28 md:pb-0">
            <div className="mx-auto w-full max-w-6xl px-4 py-10 md:py-14">
                <div className="flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
                    <div className="space-y-3">
                        <div className="text-xs uppercase tracking-[0.2em] text-base-content/60">
                            Groups
                        </div>
                        <h1 className="text-3xl font-semibold md:text-4xl">
                            Keep expenses organized
                        </h1>
                        <p className="max-w-xl text-sm text-base-content/70 md:text-base">
                            Track balances, settle up faster, and keep every
                            group in one place.
                        </p>
                    </div>
                    <div className="flex flex-col items-start gap-3 md:items-end">
                        <div className="text-sm text-base-content/70">
                            {groupCards.length} group
                            {groupCards.length === 1 ? "" : "s"}
                        </div>
                        <Link
                            to="/create_group"
                            className="btn btn-neutral w-full sm:w-auto"
                        >
                            Create Group
                        </Link>
                    </div>
                </div>

                {hasGroups ? (
                    <div className="mt-10 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
                        {groupCards.map((group) => (
                            <GroupCard key={group.id} {...group} />
                        ))}
                    </div>
                ) : (
                    <div className="mt-14 rounded-3xl border border-base-300 bg-base-100/80 p-10 text-center shadow-sm">
                        <div className="mx-auto max-w-md space-y-3">
                            <h2 className="text-xl font-semibold">
                                No groups yet
                            </h2>
                            <p className="text-sm text-base-content/70">
                                Create your first group to start splitting
                                expenses with friends.
                            </p>
                            <Link
                                to="/create_group"
                                className="btn btn-outline"
                            >
                                Start a Group
                            </Link>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
};

export default function Home() {
    return (
        <HomeProvider>
            <HomeContent />
        </HomeProvider>
    );
}
