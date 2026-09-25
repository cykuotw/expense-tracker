import { UserCapabilities } from "./capabilities";

export interface AccountSettingsData {
    nickname: string;
    firstname: string;
    lastname: string;
    email: string;
    googleConnected: boolean;
    passwordChangeAllowed: boolean;
    capabilities: UserCapabilities;
}
