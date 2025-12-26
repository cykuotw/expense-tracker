import { Navigate, Outlet } from "react-router-dom";

interface AdminGuardProps {
    isAuthenticated: boolean;
    role: string | null;
}

const AdminGuard = ({ isAuthenticated, role }: AdminGuardProps) => {
    if (!isAuthenticated) {
        return <Navigate to="/login" replace />;
    }

    if (role !== "admin") {
        return <Navigate to="/" replace />;
    }

    return <Outlet />;
};

export default AdminGuard;
