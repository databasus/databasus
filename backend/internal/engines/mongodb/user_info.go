package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// userInfo is one user's document from the usersInfo command: the roles granted to it and, when
// asked for, the privileges those roles add up to.
type userInfo struct {
	document bson.M
}

type userInfoRequest struct {
	Username         string
	AuthDatabase     string
	IsShowPrivileges bool
}

// readUserInfo runs usersInfo; a reply that lists no user returns errUserNotListed and one that
// lists something else returns errUserUnparseable, so each caller decides what an unknown user means.
func readUserInfo(ctx context.Context, client *mongo.Client, request userInfoRequest) (userInfo, error) {
	command := bson.D{
		{Key: "usersInfo", Value: bson.D{
			{Key: "user", Value: request.Username},
			{Key: "db", Value: request.AuthDatabase},
		}},
	}

	if request.IsShowPrivileges {
		command = append(command, bson.E{Key: "showPrivileges", Value: true})
	}

	var reply bson.M

	if err := client.Database(request.AuthDatabase).RunCommand(ctx, command).Decode(&reply); err != nil {
		return userInfo{}, err
	}

	users, isUserList := reply["users"].(bson.A)
	if !isUserList || len(users) == 0 {
		return userInfo{}, errUserNotListed
	}

	user, isUserDocument := users[0].(bson.M)
	if !isUserDocument {
		return userInfo{}, errUserUnparseable
	}

	return userInfo{document: user}, nil
}

func (u userInfo) hasRoleList() bool {
	_, isRoleList := u.document["roles"].(bson.A)

	return isRoleList
}

func (u userInfo) getRoles() []bson.M {
	roles, isRoleList := u.document["roles"].(bson.A)
	if !isRoleList {
		return nil
	}

	roleDocuments := make([]bson.M, 0, len(roles))

	for _, roleEntry := range roles {
		if role, isRoleDocument := roleEntry.(bson.M); isRoleDocument {
			roleDocuments = append(roleDocuments, role)
		}
	}

	return roleDocuments
}

func (u userInfo) getRoleNames() []string {
	var roleNames []string

	for _, role := range u.getRoles() {
		if roleName, _ := role["role"].(string); roleName != "" {
			roleNames = append(roleNames, roleName)
		}
	}

	return roleNames
}

func (u userInfo) getInheritedPrivileges() []bson.M {
	inheritedPrivileges, isPrivilegeList := u.document["inheritedPrivileges"].(bson.A)
	if !isPrivilegeList {
		return nil
	}

	privileges := make([]bson.M, 0, len(inheritedPrivileges))

	for _, privilegeEntry := range inheritedPrivileges {
		if privilege, isPrivilegeDocument := privilegeEntry.(bson.M); isPrivilegeDocument {
			privileges = append(privileges, privilege)
		}
	}

	return privileges
}

func getPrivilegeActions(privilege bson.M) []string {
	actions, isActionList := privilege["actions"].(bson.A)
	if !isActionList {
		return nil
	}

	actionNames := make([]string, 0, len(actions))

	for _, actionEntry := range actions {
		if action, isActionName := actionEntry.(string); isActionName {
			actionNames = append(actionNames, action)
		}
	}

	return actionNames
}
