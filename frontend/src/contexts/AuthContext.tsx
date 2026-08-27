import { ReactNode, useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { AuthContext } from "../hooks/AuthContextHooks";
import { apiFetch, setApiAuthFailureHandler } from "../lib/api";
import { UserRole } from "../types/role";

interface AuthMeResponse {
    userID?: string;
    role?: UserRole;
}

export function AuthProvider({ children }: { children: ReactNode }) {
    const navigate = useNavigate();
    const location = useLocation();
    const [loading, setLoading] = useState(true);
    const [isAuthenticated, setIsAuthenticated] = useState(false);
    const [userID, setUserID] = useState<string | null>(null);
    const [role, setRole] = useState<UserRole | null>(null);

    const clearAuthState = useCallback(() => {
        setIsAuthenticated(false);
        setUserID(null);
        setRole(null);
        setLoading(false);
    }, []);

    const refreshSession = useCallback(async () => {
        try {
            const response = await apiFetch("/auth/me", { method: "GET" });
            if (!response.ok) {
                clearAuthState();
                return false;
            }

            const data = (await response.json()) as AuthMeResponse;
            setIsAuthenticated(true);
            setUserID(data.userID ?? null);
            setRole(data.role ?? null);
            setLoading(false);
            return true;
        } catch {
            clearAuthState();
            return false;
        }
    }, [clearAuthState]);

    useEffect(() => {
        setApiAuthFailureHandler(() => {
            clearAuthState();
            if (
                location.pathname !== "/login" &&
                location.pathname !== "/register"
            ) {
                navigate("/login", { replace: true });
            }
        });

        return () => {
            setApiAuthFailureHandler(null);
        };
    }, [clearAuthState, location.pathname, navigate]);

    useEffect(() => {
        void refreshSession();
    }, [refreshSession]);

    const markLoggedIn = async () => {
        setLoading(true);
        return refreshSession();
    };

    const logout = async () => {
        try {
            await apiFetch(
                "/logout",
                {
                    method: "POST",
                },
                { authMode: "none" },
            );
        } finally {
            clearAuthState();
            navigate("/login", { replace: true });
        }
    };

    return (
        <AuthContext.Provider
            value={{
                userID,
                isAuthenticated,
                role,
                loading,
                refreshSession,
                markLoggedIn,
                logout,
            }}
        >
            {children}
        </AuthContext.Provider>
    );
}
