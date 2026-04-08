import { useEffect, useRef, useState } from "react";
import { toast } from "react-hot-toast";
import { GOOGLE_CLIENT_ID } from "../../configs/config";
import { loadGoogleAccountsId } from "../../lib/googleIdentity";

interface GoogleSignInButtonProps {
    onCredentialResponse?: (
        response: GoogleCredentialResponse,
    ) => void | Promise<void>;
}

export default function GoogleSignInButton({
    onCredentialResponse,
}: GoogleSignInButtonProps) {
    const containerRef = useRef<HTMLDivElement | null>(null);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        let cancelled = false;

        const renderGoogleButton = async () => {
            if (!containerRef.current) {
                return;
            }

            try {
                setIsLoading(true);

                const googleAccountsId = await loadGoogleAccountsId();
                if (cancelled || !containerRef.current) {
                    return;
                }

                const buttonWidth = Math.round(
                    containerRef.current.getBoundingClientRect().width || 0,
                );

                googleAccountsId.initialize({
                    client_id: GOOGLE_CLIENT_ID,
                    callback: (response) => {
                        void onCredentialResponse?.(response);
                    },
                    ux_mode: "popup",
                });

                containerRef.current.replaceChildren();
                googleAccountsId.renderButton(containerRef.current, {
                    type: "standard",
                    theme: "outline",
                    size: "large",
                    text: "continue_with",
                    shape: "pill",
                    logo_alignment: "left",
                    width: buttonWidth || undefined,
                });
            } catch (renderError) {
                if (cancelled) {
                    return;
                }

                toast.error(
                    renderError instanceof Error
                        ? renderError.message
                        : "Failed to load Google sign-in.",
                );
            } finally {
                if (!cancelled) {
                    setIsLoading(false);
                }
            }
        };

        renderGoogleButton();

        return () => {
            cancelled = true;
        };
    }, [onCredentialResponse]);

    return (
        <div className="w-full">
            <div className="mx-auto w-full max-w-md">
                <div
                    ref={containerRef}
                    className="flex min-h-11 w-full items-center justify-center"
                />
                {isLoading ? (
                    <p className="mt-3 text-center text-xs text-base-content/50">
                        Loading Google sign-in...
                    </p>
                ) : null}
            </div>
        </div>
    );
}
