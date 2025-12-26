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

    return (
        <div className="h-screen">
            <div className="flex flex-wrap justify-center items-center py-5 md:h-auto">
                {groupCards.map((group) => (
                    <GroupCard key={group.id} {...group} />
                ))}
            </div>
            <div className="py-10 block md:hidden"></div>
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
