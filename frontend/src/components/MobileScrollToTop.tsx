import { useEffect } from "react";
import { useLocation } from "react-router-dom";

export default function MobileScrollToTop() {
    const { pathname } = useLocation();

    useEffect(() => {
        const isMobile = window.matchMedia("(max-width: 767px)").matches;

        if (isMobile) {
            window.scrollTo({
                top: 0,
                left: 0,
                behavior: "auto",
            });
        }
    }, [pathname]);

    return null;
}
