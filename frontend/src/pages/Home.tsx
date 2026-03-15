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
        <div className="page-shell">
            <div className="page-container">
                <div className="page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Groups</div>
                        <h1 className="page-title">
                            Keep every expense in one calm place
                        </h1>
                        <p className="page-copy">
                            Review your active groups, jump into balances, and
                            create a new space when a trip, home, or shared tab
                            starts getting messy.
                        </p>
                    </div>
                    <div className="page-actions">
                        <div className="rounded-3xl bg-base-100/70 px-4 py-3 text-sm text-base-content/70">
                            <span className="font-semibold text-base-content stat-number">
                                {groupCards.length}
                            </span>{" "}
                            active group{groupCards.length === 1 ? "" : "s"}
                        </div>
                        <Link
                            to="/create_group"
                            className="btn btn-neutral w-full sm:w-auto"
                        >
                            Create Group
                        </Link>
                    </div>
                </div>

                <div className="grid gap-5 lg:grid-cols-[1.2fr_0.8fr]">
                    <section className="panel-card rounded-[2rem] p-6 md:p-8">
                        <div className="section-label">Overview</div>
                        <div className="mt-4 grid gap-4 sm:grid-cols-2">
                            <div className="metric-card rounded-[1.5rem] p-5">
                                <div className="text-sm text-base-content/60">
                                    Total groups
                                </div>
                                <div className="mt-3 text-3xl font-bold tracking-[-0.04em] text-primary stat-number">
                                    {groupCards.length}
                                </div>
                            </div>
                            <div className="metric-card rounded-[1.5rem] p-5">
                                <div className="text-sm text-base-content/60">
                                    Next best action
                                </div>
                                <div className="mt-3 text-lg font-semibold text-base-content">
                                    {hasGroups
                                        ? "Review unsettled balances"
                                        : "Create your first group"}
                                </div>
                            </div>
                        </div>
                    </section>
                    <aside className="panel-card rounded-[2rem] p-6 md:p-8">
                        <div className="section-label">Summary</div>
                        <h2 className="mt-3 text-2xl font-semibold tracking-[-0.03em] text-base-content">
                            Stay on top of active groups
                        </h2>
                        <p className="mt-3 text-sm leading-6 text-base-content/70">
                            Open a group to review balances, recent expenses,
                            and any members that still need to settle up.
                        </p>
                    </aside>
                </div>

                {hasGroups ? (
                    <div className="mt-8 grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
                        {groupCards.map((group) => (
                            <GroupCard key={group.id} {...group} />
                        ))}
                    </div>
                ) : (
                    <div className="panel-card mt-8 rounded-[2rem] p-10 text-center">
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
