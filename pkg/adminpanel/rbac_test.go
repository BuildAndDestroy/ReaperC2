package adminpanel

import (
	"testing"

	"ReaperC2/pkg/dbconnections"
)

func TestEffectivePortalRole(t *testing.T) {
	if effectivePortalRole(nil) != dbconnections.RoleAdmin {
		t.Fatal("nil -> admin")
	}
	if effectivePortalRole(&dbconnections.Operator{Role: ""}) != dbconnections.RoleAdmin {
		t.Fatal("empty -> admin")
	}
	if effectivePortalRole(&dbconnections.Operator{Role: "operator"}) != dbconnections.RoleOperator {
		t.Fatal("operator")
	}
	if effectivePortalRole(&dbconnections.Operator{Role: "ADMIN"}) != dbconnections.RoleAdmin {
		t.Fatal("admin case")
	}
}
