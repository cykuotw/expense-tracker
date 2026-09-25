import { UserRole } from "./role";
import { UserCapabilities } from "./capabilities";

export interface AdminUser {
    id: string;
    firstname: string;
    lastname: string;
    email: string;
    nickname: string;
    role: UserRole;
    isActive: boolean;
    isProtectedAdmin: boolean;
    createTime: string;
    capabilities: UserCapabilities;
}

export type InvitationStatus = "invited" | "expired" | "used";

export interface AdminInvitation {
    id: string;
    email: string;
    expiresAt: string;
    usedAt: string | null;
    createdAt: string;
    status: InvitationStatus;
}

export interface AdminManagementData {
    users: AdminUser[];
    invitations: AdminInvitation[];
}
