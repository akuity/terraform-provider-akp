package akp

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/suite"

	"github.com/akuity/api-client-go/pkg/api/exportfilter"
)

type ExportResourceFilterSuite struct {
	suite.Suite
}

func TestExportResourceFilterSuite(t *testing.T) {
	suite.Run(t, new(ExportResourceFilterSuite))
}

func resourceMap(keys ...string) tftypes.Map {
	elems := make(map[string]attr.Value, len(keys))
	for _, k := range keys {
		elems[k] = tftypes.StringValue("{}")
	}
	return tftypes.MapValueMust(tftypes.StringType, elems)
}

// longAppKeys returns n Application keys with 240-character names, so each
// filter entry costs 288 encoded bytes in the query string.
func longAppKeys(n int) []string {
	keys := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("argoproj.io/v1alpha1/Application/argocd/%0240d", i)
	}
	return keys
}

func (s *ExportResourceFilterSuite) TestFilter() {
	testCases := map[string]struct {
		resources tftypes.Map
		wantOK    bool
		wantRefs  []string
	}{
		"state keys become sorted entries": {
			resources: resourceMap(
				"argoproj.io/v1alpha1/AppProject/argocd/team",
				"argoproj.io/v1alpha1/Application/argocd/guestbook",
				"kargo.akuity.io/v1alpha1/Project//proj-a",
				"v1/ConfigMap/proj-a/shared",
				"admissionregistration.k8s.io/v1/ValidatingAdmissionPolicy//policy",
			),
			wantOK: true,
			wantRefs: []string{
				"/ConfigMap/proj-a/shared",
				"admissionregistration.k8s.io/ValidatingAdmissionPolicy//policy",
				"argoproj.io/AppProject/argocd/team",
				"argoproj.io/Application/argocd/guestbook",
				"kargo.akuity.io/Project//proj-a",
			},
		},
		"keys that can't match an export are dropped": {
			resources: resourceMap("my-app", "argoproj.io/v1alpha1/Application/argocd/guestbook"),
			wantOK:    true,
			wantRefs:  []string{"argoproj.io/Application/argocd/guestbook"},
		},
		// Apply accepts any key, so state can hold aliases whose parts would make the
		// server reject the whole export.
		"alias keys that aren't valid entries are dropped": {
			resources: resourceMap(
				"manifests/argocd/prod/",
				"argoproj.io/v1alpha1//argocd/app",
				"argoproj.io/v1alpha1/Application/argocd/100%",
				"argoproj.io/v1alpha1/Application/../app",
				"argoproj.io/v1alpha1/Application/argocd/guestbook",
			),
			wantOK:   true,
			wantRefs: []string{"argoproj.io/Application/argocd/guestbook"},
		},
		"null map asks for no objects":            {resources: tftypes.MapNull(tftypes.StringType), wantOK: true},
		"unknown map needs a full export":         {resources: tftypes.MapUnknown(tftypes.StringType)},
		"over the URL budget needs a full export": {resources: resourceMap(longAppKeys(15)...)},
	}
	for name, tc := range testCases {
		s.Run(name, func() {
			ok, refs := exportResourceFilter(tc.resources)
			s.Require().Equal(tc.wantOK, ok)
			s.Require().Equal(tc.wantRefs, refs)
			for _, ref := range refs {
				_, err := exportfilter.Parse(ref)
				s.Require().NoError(err, "the server would reject the whole export over %q", ref)
			}
		})
	}
}

func (s *ExportResourceFilterSuite) TestWithinURLBudget() {
	ok, refs := exportResourceFilter(resourceMap(longAppKeys(14)...))
	s.Require().True(ok)
	s.Require().Len(refs, 14)
	query := url.Values{"resources": refs}.Encode()
	s.Require().LessOrEqual(len(query), maxExportFilterQueryBytes, "the filter must fit the URL budget")
}
