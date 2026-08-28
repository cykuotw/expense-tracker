import { FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { toast } from "react-hot-toast";

import GoogleSignInButton from "../components/auth/GoogleSignInButton";
import { GOOGLE_OAUTH_ENABLED } from "../configs/config";
import { apiFetch, getResponseErrorMessage } from "../lib/api";
import { AccountSettingsData } from "../types/account";

const EMPTY_PROFILE = { firstname: "", lastname: "", nickname: "" };
const EMPTY_PASSWORDS = {
    currentPassword: "",
    newPassword: "",
    confirmPassword: "",
};
type PasswordField = keyof typeof EMPTY_PASSWORDS;

const EMPTY_PASSWORD_VISIBILITY: Record<PasswordField, boolean> = {
    currentPassword: false,
    newPassword: false,
    confirmPassword: false,
};

const PASSWORD_FIELDS: Array<{
    field: PasswordField;
    label: string;
    autoComplete: string;
    minLength?: number;
}> = [
    {
        field: "currentPassword",
        label: "Current password",
        autoComplete: "current-password",
    },
    {
        field: "newPassword",
        label: "New password",
        autoComplete: "new-password",
        minLength: 8,
    },
    {
        field: "confirmPassword",
        label: "Confirm new password",
        autoComplete: "new-password",
        minLength: 8,
    },
];
const PASSWORD_VALIDATION_DELAY_MS = 300;

type PasswordValidation = {
    valid: boolean;
    message: string;
} | null;

function passwordByteLength(value: string) {
    return new TextEncoder().encode(value).length;
}

function useDebouncedValue<T>(value: T, delay: number) {
    const [debouncedValue, setDebouncedValue] = useState(value);

    useEffect(() => {
        const timeoutID = window.setTimeout(() => setDebouncedValue(value), delay);
        return () => window.clearTimeout(timeoutID);
    }, [delay, value]);

    return debouncedValue;
}

function validateNewPassword(value: string): PasswordValidation {
    if (!value) {
        return null;
    }
    if (value.length < 8) {
        return {
            valid: false,
            message: "Password must be at least 8 characters.",
        };
    }
    if (passwordByteLength(value) > 72) {
        return {
            valid: false,
            message: "Password must be no more than 72 bytes.",
        };
    }
    return { valid: true, message: "Password length is valid." };
}

function validatePasswordConfirmation(
    newPassword: string,
    confirmation: string,
): PasswordValidation {
    if (!confirmation) {
        return null;
    }
    if (confirmation !== newPassword) {
        return { valid: false, message: "Passwords do not match." };
    }
    return { valid: true, message: "Passwords match." };
}

function validatePasswordDifference(
    currentPassword: string,
    newPassword: string,
): PasswordValidation {
    if (!currentPassword || !newPassword) {
        return null;
    }
    if (currentPassword === newPassword) {
        return {
            valid: false,
            message: "New password must be different from your current password.",
        };
    }
    return {
        valid: true,
        message: "New password is different from your current password.",
    };
}

function PasswordValidationMessage({
    id,
    validation,
}: {
    id: string;
    validation: PasswordValidation;
}) {
    return (
        <div
            id={id}
            className="mt-2 min-h-5"
            aria-live="polite"
            aria-atomic="true"
        >
            {validation ? (
                <p
                    className={`flex items-center gap-1.5 text-xs leading-5 ${validation.valid ? "text-success" : "text-error"}`}
                >
                    <svg
                        viewBox="0 0 24 24"
                        className="h-4 w-4 shrink-0"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        aria-hidden="true"
                    >
                        {validation.valid ? (
                            <path
                                d="m5 12 4 4L19 7"
                                strokeLinecap="round"
                                strokeLinejoin="round"
                            />
                        ) : (
                            <>
                                <circle cx="12" cy="12" r="9" />
                                <path
                                    d="m9 9 6 6m0-6-6 6"
                                    strokeLinecap="round"
                                />
                            </>
                        )}
                    </svg>
                    {validation.message}
                </p>
            ) : null}
        </div>
    );
}

export default function AccountSettings() {
    const [account, setAccount] = useState<AccountSettingsData | null>(null);
    const [profile, setProfile] = useState(EMPTY_PROFILE);
    const [passwords, setPasswords] = useState(EMPTY_PASSWORDS);
    const [loading, setLoading] = useState(true);
    const [loadError, setLoadError] = useState("");
    const [profileError, setProfileError] = useState("");
    const [passwordError, setPasswordError] = useState("");
    const [savingProfile, setSavingProfile] = useState(false);
    const [savingPassword, setSavingPassword] = useState(false);
    const [googleLinkExpanded, setGoogleLinkExpanded] = useState(false);
    const [googleLinkPassword, setGoogleLinkPassword] = useState("");
    const [googleLinkPasswordVisible, setGoogleLinkPasswordVisible] =
        useState(false);
    const [googleLinkError, setGoogleLinkError] = useState("");
    const [linkingGoogle, setLinkingGoogle] = useState(false);
    const googleLinkPasswordRef = useRef("");
    const [passwordVisibility, setPasswordVisibility] = useState(
        EMPTY_PASSWORD_VISIBILITY,
    );
    const debouncedCurrentPassword = useDebouncedValue(
        passwords.currentPassword,
        PASSWORD_VALIDATION_DELAY_MS,
    );
    const debouncedNewPassword = useDebouncedValue(
        passwords.newPassword,
        PASSWORD_VALIDATION_DELAY_MS,
    );
    const debouncedConfirmPassword = useDebouncedValue(
        passwords.confirmPassword,
        PASSWORD_VALIDATION_DELAY_MS,
    );

    const newPasswordValidation =
        passwords.newPassword === debouncedNewPassword
            ? validateNewPassword(debouncedNewPassword)
            : null;
    const confirmPasswordValidation =
        passwords.newPassword === debouncedNewPassword &&
        passwords.confirmPassword === debouncedConfirmPassword
            ? validatePasswordConfirmation(
                  debouncedNewPassword,
                  debouncedConfirmPassword,
              )
            : null;
    const passwordDifferenceValidation =
        passwords.currentPassword === debouncedCurrentPassword &&
        passwords.newPassword === debouncedNewPassword
            ? validatePasswordDifference(
                  debouncedCurrentPassword,
                  debouncedNewPassword,
              )
            : null;
    const isPasswordFormValid =
        newPasswordValidation?.valid === true &&
        passwordDifferenceValidation?.valid === true &&
        confirmPasswordValidation?.valid === true;

    const isProfileDirty =
        account !== null &&
        (profile.firstname !== account.firstname ||
            profile.lastname !== account.lastname ||
            profile.nickname !== account.nickname);

    const load = useCallback(async () => {
        setLoading(true);
        setLoadError("");
        try {
            const response = await apiFetch("/account");
            if (!response.ok) {
                throw new Error(
                    await getResponseErrorMessage(
                        response,
                        "Unable to load account settings",
                    ),
                );
            }
            const data = (await response.json()) as AccountSettingsData;
            setAccount(data);
            setProfile({
                firstname: data.firstname,
                lastname: data.lastname,
                nickname: data.nickname,
            });
        } catch (error) {
            setLoadError(
                error instanceof Error
                    ? error.message
                    : "Unable to load account settings",
            );
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        void load();
    }, [load]);

    const updateProfile = async (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        if (!isProfileDirty) {
            return;
        }
        const payload = {
            firstname: profile.firstname.trim(),
            lastname: profile.lastname.trim(),
            nickname: profile.nickname.trim(),
        };
        if (!payload.firstname || !payload.lastname) {
            setProfileError("First name and last name are required.");
            return;
        }
        setSavingProfile(true);
        setProfileError("");
        try {
            const response = await apiFetch("/account", {
                method: "PATCH",
                body: JSON.stringify(payload),
            });
            if (!response.ok) {
                throw new Error(
                    await getResponseErrorMessage(
                        response,
                        "Unable to update profile",
                    ),
                );
            }
            const updated = (await response.json()) as AccountSettingsData;
            setAccount(updated);
            setProfile({
                firstname: updated.firstname,
                lastname: updated.lastname,
                nickname: updated.nickname,
            });
            toast.success("Profile updated");
        } catch (error) {
            setProfileError(
                error instanceof Error ? error.message : "Unable to update profile",
            );
        } finally {
            setSavingProfile(false);
        }
    };

    const changePassword = async (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        const newPasswordBytes = passwordByteLength(passwords.newPassword);
        if (!passwords.currentPassword) {
            setPasswordError("Enter your current password.");
            return;
        }
        if (passwords.newPassword.length < 8) {
            setPasswordError("New password must be at least 8 characters.");
            return;
        }
        if (newPasswordBytes > 72) {
            setPasswordError("New password must be no more than 72 bytes.");
            return;
        }
        if (passwords.currentPassword === passwords.newPassword) {
            setPasswordError(
                "New password must be different from your current password.",
            );
            return;
        }
        if (passwords.newPassword !== passwords.confirmPassword) {
            setPasswordError("New passwords do not match.");
            return;
        }
        setSavingPassword(true);
        setPasswordError("");
        try {
            const response = await apiFetch("/account/password", {
                method: "PATCH",
                body: JSON.stringify({
                    currentPassword: passwords.currentPassword,
                    newPassword: passwords.newPassword,
                }),
            });
            if (!response.ok) {
                throw new Error(
                    await getResponseErrorMessage(
                        response,
                        "Unable to change password",
                    ),
                );
            }
            setPasswords(EMPTY_PASSWORDS);
            setPasswordVisibility(EMPTY_PASSWORD_VISIBILITY);
            toast.success("Password changed. Other sessions were signed out.");
        } catch (error) {
            setPasswordError(
                error instanceof Error ? error.message : "Unable to change password",
            );
        } finally {
            setSavingPassword(false);
        }
    };

    const resetGoogleLink = () => {
        googleLinkPasswordRef.current = "";
        setGoogleLinkPassword("");
        setGoogleLinkPasswordVisible(false);
        setGoogleLinkError("");
        setGoogleLinkExpanded(false);
    };

    const handleGoogleLinkCredential = useCallback(
        async (response: GoogleCredentialResponse) => {
            const credential = response.credential?.trim();
            if (!credential) {
                setGoogleLinkError(
                    "Google did not return a credential. Please try again.",
                );
                return;
            }

            const currentPassword = googleLinkPasswordRef.current;
            if (!currentPassword) {
                setGoogleLinkError("Enter your current password first.");
                return;
            }

            setLinkingGoogle(true);
            setGoogleLinkError("");
            try {
                const linkResponse = await apiFetch(
                    "/account/google/link",
                    {
                        method: "POST",
                        headers: {
                            Authorization: `Bearer ${credential}`,
                        },
                        body: JSON.stringify({ currentPassword }),
                    },
                    // A 401 here may describe the Google credential rather than
                    // the application session, so do not start a session refresh.
                    { authMode: "none" },
                );
                if (!linkResponse.ok) {
                    throw new Error(
                        await getResponseErrorMessage(
                            linkResponse,
                            "Unable to connect Google",
                        ),
                    );
                }

                const updated =
                    (await linkResponse.json()) as AccountSettingsData;
                setAccount(updated);
                googleLinkPasswordRef.current = "";
                setGoogleLinkPassword("");
                setGoogleLinkPasswordVisible(false);
                setGoogleLinkExpanded(false);
                toast.success(
                    "Google account connected. Other sessions were signed out.",
                );
            } catch (error) {
                setGoogleLinkError(
                    error instanceof Error
                        ? error.message
                        : "Unable to connect Google",
                );
            } finally {
                setLinkingGoogle(false);
            }
        },
        [],
    );

    if (loading) {
        return (
            <div className="page-shell">
                <div className="page-container">
                    <div className="panel-card flex min-h-64 items-center justify-center rounded-[2rem]" aria-label="Loading account settings">
                        <span className="loading loading-spinner loading-lg" />
                    </div>
                </div>
            </div>
        );
    }

    if (loadError || !account) {
        return (
            <div className="page-shell">
                <div className="page-container">
                    <div className="panel-card rounded-[2rem] border-error/30 p-6" role="alert">
                        <p className="text-error">{loadError || "Unable to load account settings"}</p>
                        <button type="button" className="btn btn-neutral mt-4 min-h-11" onClick={() => void load()}>
                            Try again
                        </button>
                    </div>
                </div>
            </div>
        );
    }

    const inputClass = "input input-bordered mt-2 min-h-11 w-full bg-base-100";

    return (
        <div className="page-shell">
            <div className="page-container max-w-5xl">
                <div className="page-header">
                    <div className="page-header__copy">
                        <div className="page-eyebrow">Account</div>
                        <h1 className="page-title">Settings</h1>
                        <p className="page-copy">
                            Keep your personal details and sign-in security up to date.
                        </p>
                    </div>
                </div>

                <div className="grid gap-6 lg:grid-cols-[minmax(0,1.1fr)_minmax(20rem,0.9fr)]">
                    <section className="panel-card rounded-[2rem] p-5 sm:p-7" aria-labelledby="profile-heading">
                        <div>
                            <div className="section-label">Personal information</div>
                            <h2 id="profile-heading" className="mt-2 text-xl font-semibold">Profile</h2>
                            <p className="mt-2 text-sm leading-6 text-base-content/65">
                                This information appears throughout your shared expense groups.
                            </p>
                        </div>
                        <form className="mt-6 space-y-5" onSubmit={updateProfile}>
                            <div className="grid gap-5 sm:grid-cols-2">
                                <label className="text-sm font-medium text-base-content">
                                    First name
                                    <input
                                        className={inputClass}
                                        value={profile.firstname}
                                        maxLength={100}
                                        autoComplete="given-name"
                                        onChange={(event) => setProfile((current) => ({ ...current, firstname: event.target.value }))}
                                    />
                                </label>
                                <label className="text-sm font-medium text-base-content">
                                    Last name
                                    <input
                                        className={inputClass}
                                        value={profile.lastname}
                                        maxLength={100}
                                        autoComplete="family-name"
                                        onChange={(event) => setProfile((current) => ({ ...current, lastname: event.target.value }))}
                                    />
                                </label>
                            </div>
                            <label className="block text-sm font-medium text-base-content">
                                Nickname
                                <input
                                    className={inputClass}
                                    value={profile.nickname}
                                    maxLength={100}
                                    autoComplete="nickname"
                                    onChange={(event) => setProfile((current) => ({ ...current, nickname: event.target.value }))}
                                />
                            </label>
                            <div>
                                <label htmlFor="account-email" className="block text-sm font-medium text-base-content">
                                    Email
                                </label>
                                <input
                                    id="account-email"
                                    type="email"
                                    className={`${inputClass} cursor-not-allowed opacity-70`}
                                    value={account.email}
                                    readOnly
                                    aria-describedby="email-help"
                                />
                                <span id="email-help" className="mt-2 block text-xs leading-5 text-base-content/60">
                                    Email changes require a separate verification flow and are not available yet.
                                </span>
                            </div>
                            {profileError ? <p className="text-sm text-error" role="alert">{profileError}</p> : null}
                            <button
                                type="submit"
                                className="btn btn-neutral min-h-11 w-full sm:w-auto"
                                disabled={savingProfile || !isProfileDirty}
                            >
                                {savingProfile ? <span className="loading loading-spinner loading-sm" /> : null}
                                {savingProfile ? "Saving…" : "Save profile"}
                            </button>
                        </form>
                    </section>

                    <section className="panel-card rounded-[2rem] p-5 sm:p-7" aria-labelledby="security-heading">
                        <div className="section-label">Sign-in security</div>
                        <h2 id="security-heading" className="mt-2 text-xl font-semibold">Password</h2>
                        {account.passwordChangeAllowed ? (
                            <>
                                <p className="mt-2 text-sm leading-6 text-base-content/65">
                                    Changing your password signs out your other sessions while keeping this device connected.
                                </p>
                                <form className="mt-6 space-y-5" onSubmit={changePassword}>
                                    {PASSWORD_FIELDS.map(({ field, label, autoComplete, minLength }) => {
                                        const isVisible = passwordVisibility[field];
                                        const inputID = `account-${field}`;
                                        const validation =
                                            field === "newPassword"
                                                ? newPasswordValidation
                                                : field === "confirmPassword"
                                                  ? confirmPasswordValidation
                                                  : null;
                                        const validationID =
                                            field === "newPassword"
                                                ? "new-password-validation"
                                                : field === "confirmPassword"
                                                  ? "confirm-password-validation"
                                                  : undefined;
                                        const describedBy =
                                            field === "newPassword"
                                                ? `password-guidance ${validationID} password-difference-validation`
                                                : validationID;
                                        return (
                                            <div key={field}>
                                                <label
                                                    htmlFor={inputID}
                                                    className="block text-sm font-medium text-base-content"
                                                >
                                                    {label}
                                                </label>
                                                <div className="relative mt-2">
                                                    <input
                                                        id={inputID}
                                                        type={isVisible ? "text" : "password"}
                                                        className="input input-bordered min-h-11 w-full bg-base-100 pr-20"
                                                        value={passwords[field]}
                                                        autoComplete={autoComplete}
                                                        minLength={minLength}
                                                        aria-describedby={describedBy}
                                                        aria-invalid={
                                                            validation?.valid === false ||
                                                            (field === "newPassword" &&
                                                                passwordDifferenceValidation?.valid === false)
                                                                ? true
                                                                : undefined
                                                        }
                                                        onChange={(event) =>
                                                            setPasswords((current) => ({
                                                                ...current,
                                                                [field]: event.target.value,
                                                            }))
                                                        }
                                                    />
                                                    <button
                                                        type="button"
                                                        className="btn btn-ghost absolute inset-y-0 right-0 min-h-11 min-w-16 rounded-l-none px-3 text-xs"
                                                        aria-label={`${isVisible ? "Hide" : "Show"} ${label.toLowerCase()}`}
                                                        aria-pressed={isVisible}
                                                        onClick={() =>
                                                            setPasswordVisibility((current) => ({
                                                                ...current,
                                                                [field]: !current[field],
                                                            }))
                                                        }
                                                    >
                                                        {isVisible ? "Hide" : "Show"}
                                                    </button>
                                                </div>
                                                {validationID ? (
                                                    <PasswordValidationMessage
                                                        id={validationID}
                                                        validation={validation}
                                                    />
                                                ) : null}
                                                {field === "newPassword" ? (
                                                    <PasswordValidationMessage
                                                        id="password-difference-validation"
                                                        validation={passwordDifferenceValidation}
                                                    />
                                                ) : null}
                                            </div>
                                        );
                                    })}
                                    <p id="password-guidance" className="text-xs leading-5 text-base-content/60">
                                        Use at least 8 characters, stay within 72 bytes, and choose a password you do not use elsewhere.
                                    </p>
                                    {passwordError ? <p className="text-sm text-error" role="alert">{passwordError}</p> : null}
                                    <button
                                        type="submit"
                                        className="btn btn-neutral min-h-11 w-full"
                                        disabled={savingPassword || !isPasswordFormValid}
                                    >
                                        {savingPassword ? <span className="loading loading-spinner loading-sm" /> : null}
                                        {savingPassword ? "Updating…" : "Change password"}
                                    </button>
                                </form>
                            </>
                        ) : (
                            <div className="mt-5 rounded-2xl border border-base-300 bg-base-200/60 p-4">
                                <div className="font-medium text-base-content">Managed by Google</div>
                                <p className="mt-2 text-sm leading-6 text-base-content/65">
                                    This account was created with Google and does not have a local password.
                                </p>
                            </div>
                        )}
                        <div className="mt-6 border-t border-base-300 pt-5">
                            <div className="flex items-center justify-between gap-4">
                                <span className="text-sm font-medium">Google account</span>
                                <span className={`badge ${account.googleConnected ? "badge-success" : "badge-ghost"}`}>
                                    {account.googleConnected ? "Connected" : "Not connected"}
                                </span>
                            </div>
                            {account.googleConnected ? (
                                <p className="mt-2 text-xs leading-5 text-base-content/60">
                                    {account.passwordChangeAllowed
                                        ? "You can sign in with either Google or your password."
                                        : "Google is the sign-in method for this account."}
                                </p>
                            ) : GOOGLE_OAUTH_ENABLED &&
                              account.passwordChangeAllowed ? (
                                <div className="mt-3">
                                    <p className="text-xs leading-5 text-base-content/60">
                                        Connect the Google account for {account.email}. You’ll confirm your password before choosing the matching Google account.
                                    </p>
                                    {!googleLinkExpanded ? (
                                        <button
                                            type="button"
                                            className="btn btn-outline mt-4 min-h-11 w-full"
                                            aria-expanded="false"
                                            aria-controls="google-link-panel"
                                            onClick={() => {
                                                setGoogleLinkError("");
                                                setGoogleLinkExpanded(true);
                                            }}
                                        >
                                            Connect Google
                                        </button>
                                    ) : (
                                        <div
                                            id="google-link-panel"
                                            className="mt-4 rounded-2xl border border-base-300 bg-base-200/50 p-4"
                                            role="region"
                                            aria-label="Connect Google account"
                                        >
                                            <label
                                                htmlFor="google-link-password"
                                                className="block text-sm font-medium text-base-content"
                                            >
                                                Confirm current password
                                            </label>
                                            <div className="relative mt-2">
                                                <input
                                                    id="google-link-password"
                                                    type={
                                                        googleLinkPasswordVisible
                                                            ? "text"
                                                            : "password"
                                                    }
                                                    className="input input-bordered min-h-11 w-full bg-base-100 pr-20"
                                                    value={googleLinkPassword}
                                                    autoComplete="current-password"
                                                    disabled={linkingGoogle}
                                                    aria-describedby="google-link-help google-link-error"
                                                    onChange={(event) => {
                                                        const value =
                                                            event.target.value;
                                                        googleLinkPasswordRef.current =
                                                            value;
                                                        setGoogleLinkPassword(value);
                                                        setGoogleLinkError("");
                                                    }}
                                                />
                                                <button
                                                    type="button"
                                                    className="btn btn-ghost absolute inset-y-0 right-0 min-h-11 min-w-16 rounded-l-none px-3 text-xs"
                                                    aria-label={`${googleLinkPasswordVisible ? "Hide" : "Show"} linking password`}
                                                    aria-pressed={
                                                        googleLinkPasswordVisible
                                                    }
                                                    disabled={linkingGoogle}
                                                    onClick={() =>
                                                        setGoogleLinkPasswordVisible(
                                                            (visible) => !visible,
                                                        )
                                                    }
                                                >
                                                    {googleLinkPasswordVisible
                                                        ? "Hide"
                                                        : "Show"}
                                                </button>
                                            </div>
                                            <p
                                                id="google-link-help"
                                                className="mt-2 text-xs leading-5 text-base-content/60"
                                            >
                                                Your Google email must match {account.email}. Other sessions will be signed out after linking.
                                            </p>
                                            <div
                                                id="google-link-error"
                                                className="mt-2 min-h-5"
                                                aria-live="polite"
                                            >
                                                {googleLinkError ? (
                                                    <p
                                                        className="text-xs leading-5 text-error"
                                                        role="alert"
                                                    >
                                                        {googleLinkError}
                                                    </p>
                                                ) : null}
                                            </div>
                                            <div
                                                className={`mt-3 ${linkingGoogle ? "pointer-events-none opacity-60" : ""}`}
                                                aria-busy={linkingGoogle}
                                            >
                                                {googleLinkPassword ? (
                                                    <GoogleSignInButton
                                                        onCredentialResponse={
                                                            handleGoogleLinkCredential
                                                        }
                                                    />
                                                ) : (
                                                    <button
                                                        type="button"
                                                        className="btn min-h-11 w-full"
                                                        disabled
                                                    >
                                                        Continue with Google
                                                    </button>
                                                )}
                                            </div>
                                            <button
                                                type="button"
                                                className="btn btn-ghost mt-2 min-h-11 w-full"
                                                disabled={linkingGoogle}
                                                onClick={resetGoogleLink}
                                            >
                                                Cancel
                                            </button>
                                        </div>
                                    )}
                                </div>
                            ) : (
                                <p className="mt-2 text-xs leading-5 text-base-content/60">
                                    Google linking is unavailable for this account.
                                </p>
                            )}
                        </div>
                    </section>
                </div>
            </div>
        </div>
    );
}
