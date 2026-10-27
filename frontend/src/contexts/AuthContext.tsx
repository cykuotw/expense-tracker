import { ReactNode, useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { AuthContext } from "../hooks/AuthContextHooks";
import { apiFetch, setApiAuthFailureHandler } from "../lib/api";
import { setServiceUnavailable, useServiceUnavailable } from "../lib/serviceAvailability";
import { UserRole } from "../types/role";

interface AuthMeResponse {
    userID?: string;
    role?: UserRole;
}

export function AuthProvider({ children }: { children: ReactNode }) {
    const navigate = useNavigate();
    const location = useLocation();
    const serviceUnavailable = useServiceUnavailable();
    const [loading, setLoading] = useState(true);
    const [isAuthenticated, setIsAuthenticated] = useState(false);
    const [isOffline, setIsOffline] = useState(() => !navigator.onLine);
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
            if (!response.ok && response.status !== 401 && response.status !== 403) {
                setServiceUnavailable(true);
                setLoading(false);
                return false;
            }
            if (!response.ok) {
                setIsOffline(false);
                clearAuthState();
                return false;
            }

            const data = (await response.json()) as AuthMeResponse;
            setServiceUnavailable(false);
            setIsAuthenticated(true);
            setUserID(data.userID ?? null);
            setRole(data.role ?? null);
            setIsOffline(false);
            setLoading(false);
            return true;
        } catch {
            setServiceUnavailable(true);
            setIsOffline(!navigator.onLine);
            setLoading(false);
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

    useEffect(() => {
        const handleOffline = () => {
            setIsOffline(true);
            setLoading(false);
        };
        const handleOnline = () => {
            void refreshSession();
        };

        window.addEventListener("offline", handleOffline);
        window.addEventListener("online", handleOnline);
        return () => {
            window.removeEventListener("offline", handleOffline);
            window.removeEventListener("online", handleOnline);
        };
    }, [refreshSession]);

    useEffect(() => {
        if (!serviceUnavailable) return;
        let attempts = 0;
        const timer = window.setInterval(() => {
            if (document.visibilityState === "visible" && attempts < 3) {
                attempts++;
                void refreshSession();
            }
            if (attempts >= 3) window.clearInterval(timer);
        }, 15_000);
        const foreground = () => {
            if (document.visibilityState === "visible") void refreshSession();
        };
        document.addEventListener("visibilitychange", foreground);
        return () => { window.clearInterval(timer); document.removeEventListener("visibilitychange", foreground); };
    }, [serviceUnavailable, refreshSession]);

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
                isOffline,
                refreshSession,
                markLoggedIn,
                logout,
            }}
        >
            {children}
        </AuthContext.Provider>
    );
}
