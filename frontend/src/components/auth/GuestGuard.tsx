import { Navigate, Outlet } from "react-router-dom";

const GuestGuard = ({ isAuthenticated }: { isAuthenticated: boolean }) => {
    return isAuthenticated ? <Navigate to="/" replace /> : <Outlet />;
};

export default GuestGuard;
