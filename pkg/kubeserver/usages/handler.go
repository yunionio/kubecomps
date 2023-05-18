package usages

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func AddUsageHandler(prefix string, app *appsrv.Application) {
	prefix = fmt.Sprintf("%s/usages", prefix)
	app.AddHandler2("GET", prefix, auth.Authenticate(ReportUsage), nil, "get_usage", nil)
}

func ReportUsage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	ownerId, scope, err, _ := db.FetchUsageOwnerScope(ctx, userCred, query)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	usage, err := DoReportUsage(ctx, scope, ownerId, query.(*jsonutils.JSONDict))
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	usageJson := jsonutils.Marshal(usage)
	body := jsonutils.NewDict()
	body.Add(usageJson, "usage")
	appsrv.SendJSON(w, body)
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/api/usages
// +onecloud:swagger-gen-route-tag=usage
// +onecloud:swagger-gen-param-query-index=3
// +onecloud:swagger-gen-resp-index=0

// report k8s cluster usages
func DoReportUsage(ctx context.Context, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, query *jsonutils.JSONDict) (*api.GlobalUsage, error) {
	usage := new(api.GlobalUsage)
	getUsage := func(scope rbacscope.TRbacScope) (*api.UsageResult, error) {
		isSystem := jsonutils.QueryBoolean(query, "is_system", false)
		clsUsage, err := models.ClusterManager.Usage(scope, ownerId, isSystem)
		if err != nil {
			return nil, errors.Wrapf(err, "get scope %s usage", scope)
		}
		ret := new(api.UsageResult)
		ret.ClusterUsage = clsUsage
		return ret, nil
	}
	// system all usage
	if scope == rbacscope.ScopeSystem {
		adminUsage, err := getUsage(scope)
		if err != nil {
			return nil, err
		}
		usage.AllUsage = adminUsage
	}
	// domain usage
	if scope.HigherThan(rbacscope.ScopeDomain) {
		domainUsage, err := getUsage(scope)
		if err != nil {
			return nil, err
		}
		usage.DomainUsage = domainUsage
	}
	// project usage
	if scope.HigherEqual(rbacscope.ScopeProject) {
		projectUsage, err := getUsage(scope)
		if err != nil {
			return nil, err
		}
		usage.ProjectUsage = projectUsage
	}
	return usage, nil
}
