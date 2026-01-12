import { createContext, useContext, MouseEvent } from "react";
import { GroupInfo } from "../types/group";
import { BalanceData } from "../types/balance";
import { ExpenseData } from "../types/expense";

export interface GroupDetailContextType {
    groupinfo: GroupInfo | null;
    balance: BalanceData | null;
    unsettledExpenses: ExpenseData[];
    unsettledLoading: boolean;
    unsettledHasMore: boolean;
    settledExpenses: ExpenseData[];
    settledLoading: boolean;
    settledHasMore: boolean;
    loading: boolean;
    groupId: string | undefined;
    handleSettle: (e: MouseEvent<HTMLButtonElement>) => Promise<void>;
    loadMoreUnsettledExpenses: () => Promise<void>;
    loadSettledExpenses: () => Promise<void>;
    loadMoreSettledExpenses: () => Promise<void>;
}

export const GroupDetailContext = createContext<
    GroupDetailContextType | undefined
>(undefined);

export const useGroupDetail = () => {
    const context = useContext(GroupDetailContext);
    if (!context) {
        throw new Error(
            "useGroupDetail must be used within a GroupDetailProvider"
        );
    }
    return context;
};
