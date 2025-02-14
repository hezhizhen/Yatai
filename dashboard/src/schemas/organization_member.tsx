import { IBaseSchema } from './base'
import { MemberRole } from './member'
import { IOrganizationSchema } from './organization'
import { IUserSchema } from './user'

export interface IOrganizationMemberSchema extends IBaseSchema {
    role: MemberRole
    creator?: IUserSchema
    user: IUserSchema
    organization: IOrganizationSchema
}
