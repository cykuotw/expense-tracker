import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "../../hooks/AuthContextHooks";
import { USER_ROLES } from "../../types/role";

const AdminGuard = () => {
    const { isAuthenticated, role } = useAuth();
    if (!isAuthenticated) {
        return <Navigate to="/login" replace />;
    }

    if (role !== USER_ROLES.admin) {
        return <Navigate to="/" replace />;
    }

    return <Outlet />;
};

export default AdminGuard;
