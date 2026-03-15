import { Navigate, Outlet } from "react-router-dom";
import { USER_ROLES, UserRole } from "../../types/role";

interface AdminGuardProps {
    isAuthenticated: boolean;
    role: UserRole | null;
}

const AdminGuard = ({ isAuthenticated, role }: AdminGuardProps) => {
    if (!isAuthenticated) {
        return <Navigate to="/login" replace />;
    }

    if (role !== USER_ROLES.admin) {
        return <Navigate to="/" replace />;
    }

    return <Outlet />;
};

export default AdminGuard;
