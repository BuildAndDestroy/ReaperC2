package adminpanel

import (
	"strings"

	"ReaperC2/pkg/dbconnections"
)

// effectivePortalRole normalizes the operator role. Missing or legacy documents default to admin.
func effectivePortalRole(op *dbconnections.Operator) string {
	if op == nil {
		return dbconnections.RoleAdmin
	}
	switch strings.ToLower(strings.TrimSpace(op.Role)) {
	case dbconnections.RoleOperator:
		return dbconnections.RoleOperator
	case dbconnections.RoleAdmin, "":
		return dbconnections.RoleAdmin
	default:
		return dbconnections.RoleAdmin
	}
}

func isAdmin(role string) bool {
	return role == dbconnections.RoleAdmin
}
