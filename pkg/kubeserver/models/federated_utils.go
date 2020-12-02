package models

import (
	"context"
	"reflect"

	"golang.org/x/sync/errgroup"
	rbac "k8s.io/api/rbac/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

func GetFedRunningCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) ([]SCluster, error) {
	// TODO: support specify domain
	q := GetClusterManager().Query().Equals("status", api.ClusterStatusRunning)
	if scope, _ := query.GetString("scope"); scope != string(rbacutils.ScopeSystem) {
		q = q.Equals("domain_id", userCred.GetDomainId())
	}
	clusters := make([]SCluster, 0)
	if err := db.FetchModelObjects(GetClusterManager(), q, &clusters); err != nil {
		return nil, errors.Wrapf(err, "get domain %s running clusters", userCred.GetDomainId())
	}
	return clusters, nil
}

func callFedClustersGetResFunc(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	gf func(ctx context.Context, userCred mcclient.TokenCredential, c *SCluster, query jsonutils.JSONObject) (api.IClusterRemoteResources, error),
) (api.IClusterRemoteResources, error) {
	fetcher := func() ([]api.IClusterRemoteResources, error) {
		clusters, err := GetFedRunningCluster(ctx, userCred, query)
		if err != nil {
			return nil, err
		}
		if len(clusters) == 0 {
			return nil, nil
		}

		g, ctx := errgroup.WithContext(ctx)
		results := make([]api.IClusterRemoteResources, len(clusters))
		for i := range clusters {
			c := clusters[i]
			g.Go(func() error {
				resources, err := gf(ctx, userCred, &c, query)
				if err != nil {
					return err
				}
				results[i] = resources
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
		return results, nil
	}
	var resources api.IClusterRemoteResources
	allResources, err := fetcher()
	if err != nil {
		return nil, errors.Wrap(err, "fetch clusters api resources")
	}
	fRess := make([]api.IClusterRemoteResources, 0)
	for _, ress := range allResources {
		if ress == nil || gotypes.IsNil(ress) {
			continue
		}
		fRess = append(fRess, ress)
	}
	if len(fRess) == 0 {
		return nil, nil
	}

	resources = fRess[0]
	if len(fRess) == 1 {
		return resources, nil
	}
	for _, res := range fRess[1 : len(fRess)-1] {
		resources = resources.Unionset(res)
	}
	return resources, nil
}

func GetFedClustersApiResources(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.IClusterRemoteResources, error) {
	return callFedClustersGetResFunc(ctx, userCred, query,
		func(ctx context.Context, userCred mcclient.TokenCredential, c *SCluster, query jsonutils.JSONObject) (api.IClusterRemoteResources, error) {
			return c.GetDetailsApiResources(ctx, userCred, query)
		})
}

func GetFedClustersUsers(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.IClusterRemoteResources, error) {
	return callFedClustersGetResFunc(ctx, userCred, query,
		func(ctx context.Context, userCred mcclient.TokenCredential, c *SCluster, query jsonutils.JSONObject) (api.IClusterRemoteResources, error) {
			return c.GetDetailsClusterUsers(ctx, userCred, query)
		})
}

func GetFedClustersUserGroups(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.IClusterRemoteResources, error) {
	return callFedClustersGetResFunc(ctx, userCred, query,
		func(ctx context.Context, userCred mcclient.TokenCredential, c *SCluster, query jsonutils.JSONObject) (api.IClusterRemoteResources, error) {
			return c.GetDetailsClusterUserGroups(ctx, userCred, query)
		})
}

func ValidateFederatedRoleRef(ctx context.Context, userCred mcclient.TokenCredential, roleRef rbac.RoleRef) error {
	roleKind := roleRef.Kind
	roleName := roleRef.Name
	var man IFedModelManager
	if roleKind == api.KindNameClusterRole {
		man = GetFedClusterRoleManager()
	} else if roleKind == api.KindNameRole {
		man = GetFedRoleManager()
	} else {
		return httperrors.NewInputParameterError("Not support role kind %s", roleKind)
	}
	_, err := man.FetchByName(userCred, roleName)
	if err != nil {
		return errors.Wrapf(err, "Not found federated %s role object by name: %s", roleKind, roleName)
	}
	return nil
}

// GetObjectPtr wraps the given value with pointer: V => *V, *V => **V, etc.
func GetObjectPtr(obj interface{}) interface{} {
	v := reflect.ValueOf(obj)
	pt := reflect.PtrTo(v.Type())
	pv := reflect.New(pt.Elem())
	pv.Elem().Set(v)
	return pv.Interface()
}
