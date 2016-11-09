// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build integration

package facade

import (
	"strings"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk/registry"

	"errors"
	"fmt"

	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

var (
	ErrTestEPValidationFail = errors.New("Endpoint failed validation")
)

func (ft *FacadeTest) TestFacade_validateServiceName(c *C) {
	svcA := service.Service{
		ID:           "validate-service-name-A",
		Name:         "TestFacade_validateServiceNameA",
		DeploymentID: "deployment-id",
		PoolID:       "pool-id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcA), IsNil)
	svcB := service.Service{
		ID:              "validate-service-name-B",
		ParentServiceID: "validate-service-name-A",
		Name:            "TestFacade_validateServiceNameB",
		DeploymentID:    "deployment-id",
		PoolID:          "pool-id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcB), IsNil)
	// parent not exist
	err := ft.Facade.validateServiceName(ft.CTX, &service.Service{
		ID:              "validate-service-name-C",
		ParentServiceID: "bogus-parent",
		Name:            "TestFacade_validateServiceNameB",
		DeploymentID:    "deployment-id",
		PoolID:          "pool-id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
	})
	c.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
	// collision
	err = ft.Facade.validateServiceName(ft.CTX, &service.Service{
		ID:              "validate-service-name-C",
		ParentServiceID: "validate-service-name-A",
		Name:            "TestFacade_validateServiceNameB",
		DeploymentID:    "deployment-id",
		PoolID:          "pool-id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
	})
	c.Assert(err, Equals, ErrServiceCollision)
	// success
	err = ft.Facade.validateServiceName(ft.CTX, &service.Service{
		ID:              "validate-service-name-C",
		ParentServiceID: "validate-service-name-A",
		Name:            "TestFacade_validateServiceNameC",
		DeploymentID:    "deployment-id",
		PoolID:          "pool-id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
	})
	c.Assert(err, IsNil)
}

func (ft *FacadeTest) TestFacade_validateServiceTenant(c *C) {
	svcA := service.Service{
		ID:           "validate-service-tenant-A",
		Name:         "TestFacade_validateServiceTenantA",
		DeploymentID: "deployment-id",
		PoolID:       "pool-id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcA), IsNil)
	svcB := service.Service{
		ID:              "validate-service-tenant-B",
		ParentServiceID: "validate-service-tenant-A",
		Name:            "TestFacade_validateServiceTenantA",
		DeploymentID:    "deployment-id",
		PoolID:          "pool-id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcB), IsNil)
	svcC := service.Service{
		ID:           "validate-service-tenant-C",
		Name:         "TestFacade_validateServiceTenantC",
		DeploymentID: "deployment-id",
		PoolID:       "pool-id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcC), IsNil)
	// empty tenant field
	err := ft.Facade.validateServiceTenant(ft.CTX, "", "")
	c.Assert(err, Equals, ErrTenantDoesNotMatch)
	err = ft.Facade.validateServiceTenant(ft.CTX, svcA.ID, "")
	c.Assert(err, Equals, ErrTenantDoesNotMatch)
	err = ft.Facade.validateServiceTenant(ft.CTX, "", svcB.ID)
	c.Assert(err, Equals, ErrTenantDoesNotMatch)
	// service not found
	err = ft.Facade.validateServiceTenant(ft.CTX, "bogus-service", svcC.ID)
	c.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
	err = ft.Facade.validateServiceTenant(ft.CTX, svcA.ID, "bogus-service")
	c.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
	// not matching tenant
	err = ft.Facade.validateServiceTenant(ft.CTX, svcB.ID, svcC.ID)
	c.Assert(err, Equals, ErrTenantDoesNotMatch)
	err = ft.Facade.validateServiceTenant(ft.CTX, svcA.ID, svcB.ID)
	c.Assert(err, IsNil)
}

func (ft *FacadeTest) setup_validateServiceStart(c *C, endpoints ...service.ServiceEndpoint) *service.Service {
	err := ft.Facade.AddResourcePool(ft.CTX, &pool.ResourcePool{ID: "test-pool"})
	c.Assert(err, IsNil)
	svc := service.Service{
		ID:           "validate-service-start",
		Name:         "TestFacade_validateServiceStart",
		DeploymentID: "deployment-id",
		PoolID:       "test-pool",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
	}
	svc.Endpoints = endpoints
	c.Assert(ft.Facade.AddService(ft.CTX, svc), IsNil)
	return &svc
}

func (ft *FacadeTest) TestFacade_validateServiceStart_missingAddressAssignment(c *C) {
	// set up the endpoint with a missing address assignment
	endpoint := service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
		Name:        "ep1",
		Application: "ep1",
		Purpose:     "export",
		AddressConfig: servicedefinition.AddressResourceConfig{
			Port:     1234,
			Protocol: "tcp",
		},
	})
	svc := ft.setup_validateServiceStart(c, endpoint)
	err := ft.Facade.validateServiceStart(ft.CTX, svc)
	c.Assert(err, Equals, ErrServiceMissingAssignment)

}

func (ft *FacadeTest) TestFacade_validateServiceStart_checkVHostFail(c *C) {
	// set up the endpoint with an invalid vhost
	endpoint := service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
		Name:        "ep2",
		Application: "ep2",
		Purpose:     "export",
		VHostList: []servicedefinition.VHost{
			{
				Name:    "vh1",
				Enabled: true,
			},
		},
	})
	svc := ft.setup_validateServiceStart(c, endpoint)
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey("vh1-0"), svc.ID).Return(ErrTestEPValidationFail)
	err := ft.Facade.validateServiceStart(ft.CTX, svc)
	c.Assert(err, Equals, ErrTestEPValidationFail)
}

func (ft *FacadeTest) TestFacade_validateServiceStart_checkPortFail(c *C) {
	// set up the endpoint with in invalid public port
	endpoint := service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
		Name:        "ep3",
		Application: "ep3",
		Purpose:     "export",
		PortList: []servicedefinition.Port{
			{
				PortAddr: ":1234",
				Enabled:  true,
			},
		},
	})
	svc := ft.setup_validateServiceStart(c, endpoint)
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey(":1234-1"), svc.ID).Return(ErrTestEPValidationFail)
	err := ft.Facade.validateServiceStart(ft.CTX, svc)
	c.Assert(err, Equals, ErrTestEPValidationFail)
}

func (ft *FacadeTest) TestFacade_validateServiceStart(c *C) {
	// successfully add address assignment, vhost, and port
	ep1 := service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
		Name:        "ep1",
		Application: "ep1",
		Purpose:     "export",
		AddressConfig: servicedefinition.AddressResourceConfig{
			Port:     1234,
			Protocol: "tcp",
		},
	})
	ep2 := service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
		Name:        "ep2",
		Application: "ep2",
		Purpose:     "export",
		VHostList: []servicedefinition.VHost{
			{
				Name:    "vh1",
				Enabled: true,
			},
		},
	})
	ep3 := service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
		Name:        "ep3",
		Application: "ep3",
		Purpose:     "export",
		PortList: []servicedefinition.Port{
			{
				PortAddr: ":1234",
				Enabled:  true,
			},
		},
	})
	svc := ft.setup_validateServiceStart(c, ep1, ep2, ep3)
	// set up an address assignment for ep1
	err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{
		PoolID:        svc.PoolID,
		IP:            "192.168.22.12",
		Netmask:       "255.255.255.0",
		BindInterface: "eth0",
	})
	c.Assert(err, IsNil)
	err = ft.Facade.AssignIPs(ft.CTX, dao.AssignmentRequest{
		ServiceID:      svc.ID,
		AutoAssignment: false,
		IPAddress:      "192.168.22.12",
	})
	c.Assert(err, IsNil)
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey("vh1-0"), svc.ID).Return(nil)
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey(":1234-1"), svc.ID).Return(nil)
	err = ft.Facade.validateServiceStart(ft.CTX, svc)
	c.Assert(err, IsNil)
}

func (ft *FacadeTest) TestFacade_validateService_badServiceID(t *C) {
	_, err := ft.Facade.validateServiceUpdate(ft.CTX, &service.Service{ID: "badID"})
	t.Assert(err, ErrorMatches, "No such entity {kind:service, id:badID}")
}

func (ft *FacadeTest) TestFacade_validateServiceEndpoints_noDupsInOneService(t *C) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_validateServiceEndpoints",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
		},
	}

	err := ft.Facade.validateServiceEndpoints(ft.CTX, &svc)
	t.Assert(err, IsNil)
}

func (ft *FacadeTest) TestFacade_validateServiceEndpoints_noDupsInAllServices(t *C) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_validateServiceEndpoints",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
		},
	}

	if err := ft.Facade.AddService(ft.CTX, svc); err != nil {
		t.Fatalf("Setup failed; could not add svc %s: %s", svc.ID, err)
		return
	}

	childSvc := service.Service{
		ID:              "svc2",
		ParentServiceID: svc.ID,
		Name:            "TestFacade_validateServiceEndpoints_child",
		DeploymentID:    "deployment_id",
		PoolID:          "pool_id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_3", Application: "test_ep_3", Purpose: "export"}),
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_4", Application: "test_ep_4", Purpose: "export"}),
		},
	}
	if err := ft.Facade.AddService(ft.CTX, childSvc); err != nil {
		t.Fatalf("Setup failed; could not add svc %s: %s", childSvc.ID, err)
		return
	}

	err := ft.Facade.validateServiceEndpoints(ft.CTX, &svc)
	t.Assert(err, IsNil)
}

func (ft *FacadeTest) TestFacade_validateServiceEndpoints_dupsInOneService(t *C) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_validateServiceEndpoints",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
		},
	}

	err := ft.Facade.validateServiceEndpoints(ft.CTX, &svc)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)
}

func (ft *FacadeTest) TestFacade_validateServiceEndpoints_dupsBtwnServices(t *C) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_validateServiceEndpoints",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
		},
	}

	if err := ft.Facade.AddService(ft.CTX, svc); err != nil {
		t.Fatalf("Setup failed; could not add svc %s: %s", svc.ID, err)
		return
	}

	childSvc := service.Service{
		ID:              "svc2",
		ParentServiceID: svc.ID,
		Name:            "TestFacade_validateServiceEndpoints_child",
		DeploymentID:    "deployment_id",
		PoolID:          "pool_id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
		},
	}
	if err := ft.Facade.AddService(ft.CTX, childSvc); err != nil {
		t.Fatalf("Setup failed; could not add svc %s: %s", childSvc.ID, err)
		return
	}

	err := ft.Facade.validateServiceEndpoints(ft.CTX, &svc)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)
}

func (ft *FacadeTest) TestFacade_migrateServiceConfigs_noConfigs(t *C) {
	_, newSvc, err := ft.setupMigrationServices(t, nil)
	t.Assert(err, IsNil)

	err = ft.Facade.MigrateService(ft.CTX, *newSvc)
	t.Assert(err, IsNil)
}

func (ft *FacadeTest) TestFacade_migrateServiceConfigs_noChanges(t *C) {
	_, newSvc, err := ft.setupMigrationServices(t, getOriginalConfigs())
	t.Assert(err, IsNil)

	err = ft.Facade.MigrateService(ft.CTX, *newSvc)
	t.Assert(err, IsNil)
}

// Verify migration of configuration data when the user has not changed any config files
func (ft *FacadeTest) TestFacade_migrateService_withoutUserConfigChanges(t *C) {
	_, newSvc, err := ft.setupMigrationServices(t, getOriginalConfigs())
	t.Assert(err, IsNil)
	newSvc.ConfigFiles = nil

	err = ft.Facade.MigrateService(ft.CTX, *newSvc)
	t.Assert(err, IsNil)

	result, err := ft.Facade.GetService(ft.CTX, newSvc.ID)
	t.Assert(err, IsNil)

	t.Assert(result.Description, Equals, newSvc.Description)
	t.Assert(result.OriginalConfigs, DeepEquals, newSvc.OriginalConfigs)
	t.Assert(result.ConfigFiles, DeepEquals, newSvc.OriginalConfigs)

	confs, err := ft.getConfigFiles(result)
	t.Assert(err, IsNil)
	t.Assert(len(confs), Equals, 0)
}

func (ft *FacadeTest) TestFacade_GetServiceEndpoints_UndefinedService(t *C) {
	endpointMap, err := ft.Facade.GetServiceEndpoints(ft.CTX, "undefined", true, true, true)

	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "Could not find service undefined.*")
	t.Assert(endpointMap, IsNil)
}

func (ft *FacadeTest) TestFacade_GetServiceEndpoints_ZKUnavailable(t *C) {
	svc, err := ft.setupServiceWithEndpoints(t)
	t.Assert(err, IsNil)
	serviceIDs := []string{svc.ID}
	errorStub := fmt.Errorf("Stub for cannot-connect-to-zookeeper")
	ft.zzk.On("GetServiceStates", svc.PoolID, mock.AnythingOfType("*[]servicestate.ServiceState"), serviceIDs).Return(errorStub)

	endpointMap, err := ft.Facade.GetServiceEndpoints(ft.CTX, svc.ID, true, true, true)

	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "Could not get service states for service .*")
	t.Assert(endpointMap, IsNil)
}

func (ft *FacadeTest) TestFacade_GetServiceEndpoints_ServiceNotRunning(t *C) {
	svc, err := ft.setupServiceWithEndpoints(t)
	t.Assert(err, IsNil)
	serviceIDs := []string{svc.ID}
	ft.zzk.On("GetServiceStates", svc.PoolID, mock.AnythingOfType("*[]servicestate.ServiceState"), serviceIDs).Return(nil)

	endpoints, err := ft.Facade.GetServiceEndpoints(ft.CTX, svc.ID, true, true, true)

	t.Assert(err, IsNil)
	t.Assert(endpoints, NotNil)
	t.Assert(len(endpoints), Equals, 2)
	t.Assert(endpoints[0].Endpoint.ServiceID, Equals, svc.ID)
	t.Assert(endpoints[0].Endpoint.InstanceID, Equals, 0)
	t.Assert(endpoints[0].Endpoint.Application, Equals, "test_ep_1")
	t.Assert(endpoints[1].Endpoint.ServiceID, Equals, "svc1")
	t.Assert(endpoints[1].Endpoint.InstanceID, Equals, 0)
	t.Assert(endpoints[1].Endpoint.Application, Equals, "test_ep_2")
}

func (ft *FacadeTest) TestFacade_GetServiceEndpoints_ServiceRunning(t *C) {
	svc, err := ft.setupServiceWithEndpoints(t)
	t.Assert(err, IsNil)
	serviceIDs := []string{svc.ID}
	ft.zzk.On("GetServiceStates", svc.PoolID, mock.AnythingOfType("*[]servicestate.ServiceState"), serviceIDs).
		Return(nil).Run(func(args mock.Arguments) {
		// Mock results for 2 running instances
		statesArg := args.Get(1).(*[]servicestate.ServiceState)
		*statesArg = []servicestate.ServiceState{
			{ServiceID: svc.ID, InstanceID: 0, Endpoints: svc.Endpoints},
			{ServiceID: svc.ID, InstanceID: 1, Endpoints: svc.Endpoints},
		}
		t.Assert(true, Equals, true)
	})
	// don't worry about mocking the ZK validation
	ft.zzk.On("GetServiceEndpoints", svc.ID, svc.ID, mock.AnythingOfType("*[]applicationendpoint.ApplicationEndpoint")).Return(nil)

	endpoints, err := ft.Facade.GetServiceEndpoints(ft.CTX, svc.ID, true, true, true)

	t.Assert(err, IsNil)
	t.Assert(endpoints, NotNil)
	t.Assert(len(endpoints), Equals, 4)
	t.Assert(endpoints[0].Endpoint.ServiceID, Equals, svc.ID)
	t.Assert(endpoints[0].Endpoint.InstanceID, Equals, 0)
	t.Assert(endpoints[0].Endpoint.Application, Equals, "test_ep_1")
	t.Assert(endpoints[1].Endpoint.ServiceID, Equals, "svc1")
	t.Assert(endpoints[1].Endpoint.InstanceID, Equals, 0)
	t.Assert(endpoints[1].Endpoint.Application, Equals, "test_ep_2")
	t.Assert(endpoints[2].Endpoint.ServiceID, Equals, "svc1")
	t.Assert(endpoints[2].Endpoint.InstanceID, Equals, 1)
	t.Assert(endpoints[2].Endpoint.Application, Equals, "test_ep_1")
	t.Assert(endpoints[3].Endpoint.ServiceID, Equals, "svc1")
	t.Assert(endpoints[3].Endpoint.InstanceID, Equals, 1)
	t.Assert(endpoints[3].Endpoint.Application, Equals, "test_ep_2")
}

func (ft *FacadeTest) TestFacade_MigrateServices_Deploy_FailDupeEndpointsWithTemplate(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	// Try to deploy 2 services with the same parent and same templated endpoint
	deployRequest := ft.createServiceDeploymentRequest(t)
	deployRequest.ParentID = "original_service_id_child_1"
	deployRequest.Service.Endpoints = []servicedefinition.EndpointDefinition{
		servicedefinition.EndpointDefinition{
			Name:        "original_service_endpoint_name_child_1",
			Application: "{{(parent .).Name}}_original_service_endpoint_application_child_1",
			Purpose:     "export",
		},
	}

	deployRequest2 := ft.createServiceDeploymentRequest(t)
	deployRequest2.ParentID = "original_service_id_child_1"
	deployRequest2.Service.Name = "added-service-2"
	deployRequest2.Service.Endpoints = []servicedefinition.EndpointDefinition{
		servicedefinition.EndpointDefinition{
			Name:        "original_service_endpoint_name_child_1",
			Application: "{{(parent .).Name}}_original_service_endpoint_application_child_1",
			Purpose:     "export",
		},
	}

	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Deploy:    []*dao.ServiceDeploymentRequest{deployRequest, deployRequest2},
	}

	ft.dfs.On("Download",
		deployRequest.Service.ImageID,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("bool"),
	).Return("mockImageId", nil)

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, Equals, ErrServiceDuplicateEndpoint)
}

func (ft *FacadeTest) TestFacade_MigrateServices_Deploy_EndpointsWithTemplate(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	// Deploy 2 services with the same templated endpoint but different parents
	deployRequest := ft.createServiceDeploymentRequest(t)
	deployRequest.ParentID = "original_service_id_child_1"
	deployRequest.Service.Endpoints = []servicedefinition.EndpointDefinition{
		servicedefinition.EndpointDefinition{
			Name:        "original_service_endpoint_name_child_1",
			Application: "{{(parent .).Name}}_original_service_endpoint_application_child_1",
			Purpose:     "export",
		},
	}

	deployRequest2 := ft.createServiceDeploymentRequest(t)
	deployRequest2.ParentID = "original_service_id_child_0"
	deployRequest2.Service.Name = "added-service-2"
	deployRequest2.Service.Endpoints = []servicedefinition.EndpointDefinition{
		servicedefinition.EndpointDefinition{
			Name:        "original_service_endpoint_name_child_1",
			Application: "{{(parent .).Name}}_original_service_endpoint_application_child_1",
			Purpose:     "export",
		},
	}

	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Deploy:    []*dao.ServiceDeploymentRequest{deployRequest, deployRequest2},
	}

	ft.dfs.On("Download",
		deployRequest.Service.ImageID,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("bool"),
	).Return("mockImageId", nil)

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, IsNil)
}

func (ft *FacadeTest) setupServiceWithEndpoints(t *C) (*service.Service, error) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_GetServiceEndpoints",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
			service.BuildServiceEndpoint(
				servicedefinition.EndpointDefinition{
					Name: "test_ep_1", Application: "test_ep_1", Purpose: "export",
					VHostList: []servicedefinition.VHost{servicedefinition.VHost{Name: "test_vhost_1", Enabled: true}},
					PortList:  []servicedefinition.Port{servicedefinition.Port{PortAddr: ":1234", Enabled: true}},
				},
			),
		},
	}

	if err := ft.Facade.AddService(ft.CTX, svc); err != nil {
		t.Errorf("Setup failed; could not add svc %s: %s", svc.ID, err)
		return nil, err
	}
	return &svc, nil
}

func (ft *FacadeTest) setupMigrationServices(t *C, originalConfigs map[string]servicedefinition.ConfigFile) (*service.Service, *service.Service, error) {
	svc := service.Service{
		ID:              "svc1",
		Name:            "TestFacade_migrateServiceConfigs_oldSvc",
		DeploymentID:    "deployment_id",
		PoolID:          "pool_id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
		OriginalConfigs: originalConfigs,
	}

	if err := ft.Facade.AddService(ft.CTX, svc); err != nil {
		t.Errorf("Setup failed; could not add svc %s: %s", svc.ID, err)
		return nil, nil, err
	}

	oldSvc, err := ft.Facade.GetService(ft.CTX, svc.ID)
	if err != nil {
		t.Errorf("Setup failed; could not get svc %s: %s", oldSvc.ID, err)
		return nil, nil, err
	}

	newSvc := service.Service{}
	newSvc = *oldSvc
	newSvc.Description = "migrated service"

	if originalConfigs != nil {
		newSvc.OriginalConfigs = make(map[string]servicedefinition.ConfigFile)
		newSvc.OriginalConfigs["unchangedConfig"] = oldSvc.OriginalConfigs["unchangedConfig"]
		newSvc.OriginalConfigs["addedConfig"] = servicedefinition.ConfigFile{Filename: "addedConfig", Content: "original version"}

		newSvc.ConfigFiles = make(map[string]servicedefinition.ConfigFile)
		for filename, conf := range newSvc.OriginalConfigs {
			newSvc.ConfigFiles[filename] = conf
		}
	}

	return oldSvc, &newSvc, nil
}

func (ft *FacadeTest) setupConfigCustomizations(svc *service.Service) error {
	for filename, conf := range svc.OriginalConfigs {
		customizedConf := conf
		customizedConf.Content = "some user customized content"
		svc.ConfigFiles[filename] = customizedConf
	}

	err := ft.Facade.updateService(ft.CTX, *svc, false, false)
	if err != nil {
		return err
	}

	result, err := ft.Facade.GetService(ft.CTX, svc.ID)
	*svc = *result

	return err
}

func (ft *FacadeTest) getConfigFiles(svc *service.Service) ([]*serviceconfigfile.SvcConfigFile, error) {
	tenantID, servicePath, err := ft.Facade.getServicePath(ft.CTX, svc.ID)
	if err != nil {
		return nil, err
	}
	configStore := serviceconfigfile.NewStore()
	return configStore.GetConfigFiles(ft.CTX, tenantID, servicePath)
}

func getOriginalConfigs() map[string]servicedefinition.ConfigFile {
	originalConfigs := make(map[string]servicedefinition.ConfigFile)
	originalConfigs["unchangedConfig"] = servicedefinition.ConfigFile{Filename: "unchangedConfig", Content: "original version"}
	originalConfigs["deletedConfig"] = servicedefinition.ConfigFile{Filename: "deletedConfig", Content: "original version"}
	return originalConfigs
}
