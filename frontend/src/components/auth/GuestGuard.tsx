import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "../../hooks/AuthContextHooks";

const GuestGuard = () => {
    const { isAuthenticated } = useAuth();
    return isAuthenticated ? <Navigate to="/" replace /> : <Outlet />;
};

export default GuestGuard;
