export interface UserData {
    id: string;
    username: string;
    firstname: string;
    lastname: string;
    email: string;
    nickname: string;
    passwordHashed: string;
    externalType: string;
    externalId: string;
    createTime: string;
    isActive: boolean;
}
