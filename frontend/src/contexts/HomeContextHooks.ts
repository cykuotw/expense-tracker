import { createContext, useContext } from "react";
import { GroupCardData } from "../types/group";

export interface HomeContextType {
    groupCards: GroupCardData[];
    loading: boolean;
}

export const HomeContext = createContext<HomeContextType | undefined>(
    undefined
);

export const useHome = () => {
    const context = useContext(HomeContext);
    if (!context) {
        throw new Error("useHome must be used within a HomeProvider");
    }
    return context;
};
