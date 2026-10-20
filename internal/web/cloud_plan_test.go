package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_cloudPlan_isPaid(t *testing.T) {
	yes, no := true, false
	for _, tc := range []struct {
		name string
		plan cloudPlan
		want bool
	}{
		{"cloud says paid", cloudPlan{Name: "enterprise", Paid: &yes}, true},
		{"cloud says free, whatever the name", cloudPlan{Name: "pro", Paid: &no}, false},
		{"older cloud, team", cloudPlan{Name: "team"}, true},
		{"older cloud, pro", cloudPlan{Name: "pro"}, true},
		{"older cloud, free", cloudPlan{Name: "free"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.plan.isPaid())
		})
	}
}
