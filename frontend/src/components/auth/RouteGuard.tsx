import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "../../hooks/AuthContextHooks";
import { USER_ROLES } from "../../types/role";

export type RouteGuardMode = "guest" | "authenticated" | "admin";

interface RouteGuardProps {
    mode: RouteGuardMode;
}

export default function RouteGuard({ mode }: RouteGuardProps) {
    const { isAuthenticated, role } = useAuth();

    if (mode === "guest") {
        return isAuthenticated ? <Navigate to="/" replace /> : <Outlet />;
    }

    if (!isAuthenticated) {
        return <Navigate to="/login" replace />;
    }

    if (mode === "admin" && role !== USER_ROLES.admin) {
        return <Navigate to="/" replace />;
    }

    return <Outlet />;
}
