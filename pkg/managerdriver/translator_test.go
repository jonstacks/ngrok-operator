package managerdriver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	common "github.com/ngrok/ngrok-operator/api/common/v1alpha1"
	ingressv1alpha1 "github.com/ngrok/ngrok-operator/api/ingress/v1alpha1"
	ngrokv1alpha1 "github.com/ngrok/ngrok-operator/api/ngrok/v1alpha1"
	"github.com/ngrok/ngrok-operator/internal/ir"
	"github.com/ngrok/ngrok-operator/internal/testutils"
	"github.com/ngrok/ngrok-operator/internal/trafficpolicy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func stringPtr(input string) *string {
	return &input
}

func TestBuildInternalAgentEndpoint(t *testing.T) {
	testCases := []struct {
		name                   string
		irVirtualHost          *ir.IRVirtualHost
		irService              ir.IRService
		clusterDomain          string
		metadata               string
		expectedName           string
		expectedURL            string
		expectedUpstream       string
		upstreamClientCertRefs []ir.IRObjectRef
	}{
		{
			name: "Default cluster domain",
			irService: ir.IRService{
				UID:       "abc123",
				Name:      "test-service",
				Namespace: "default",
				Port:      8080,
				Scheme:    ir.IRScheme_HTTP,
			},
			clusterDomain: "cluster.local",

			irVirtualHost: &ir.IRVirtualHost{
				LabelsToAdd:      map[string]string{"label-app": "label-test"},
				AnnotationsToAdd: map[string]string{"annotation-app": "annotation-test"},
				Listener: ir.IRListener{
					Hostname: "test-hostname.ngrok.io",
					Port:     int32(443),
					Protocol: ir.IRProtocol_HTTPS,
				},
			},
			metadata:         "metadata-test",
			expectedName:     "6ca13-test-service-default-cluster.local-8080",
			expectedURL:      "https://6ca13-test-service-default-cluster-local-8080.internal",
			expectedUpstream: "http://test-service.default-cluster.local:8080",
		},
		{
			name: "Custom cluster domain",
			irService: ir.IRService{
				UID:       "xyz789",
				Name:      "another-service",
				Namespace: "custom-namespace",
				Port:      9090,
				Scheme:    ir.IRScheme_HTTP,
			},
			irVirtualHost: &ir.IRVirtualHost{
				LabelsToAdd:      map[string]string{"label-app": "label-test"},
				AnnotationsToAdd: map[string]string{"annotation-app": "annotation-test"},
				Listener: ir.IRListener{
					Hostname: "test-hostname.ngrok.io",
					Port:     int32(443),
					Protocol: ir.IRProtocol_HTTPS,
				},
			},
			clusterDomain:    "custom.domain",
			metadata:         "prod-metadata",
			expectedName:     "5a464-another-service-custom-namespace-custom.domain-9090",
			expectedURL:      "https://5a464-another-service-custom-namespace-custom-domain-9090.internal",
			expectedUpstream: "http://another-service.custom-namespace-custom.domain:9090",
		},

		{
			name: "Client cert refs",
			irService: ir.IRService{
				UID:       "xyz789",
				Name:      "another-service",
				Namespace: "custom-namespace",
				Port:      443,
				Scheme:    ir.IRScheme_HTTPS,
				ClientCertRefs: []ir.IRObjectRef{{
					Name:      "client-cert-secret",
					Namespace: "secrets",
				}},
				Protocol: ptr.To(common.ApplicationProtocol_HTTP2),
			},
			irVirtualHost: &ir.IRVirtualHost{
				LabelsToAdd:      map[string]string{"label-app": "label-test"},
				AnnotationsToAdd: map[string]string{"annotation-app": "annotation-test"},
				Listener: ir.IRListener{
					Hostname: "test-hostname.ngrok.io",
					Port:     int32(443),
					Protocol: ir.IRProtocol_HTTPS,
				},
			},
			clusterDomain:    "custom.domain",
			metadata:         "prod-metadata",
			expectedName:     "5a464-another-service-custom-namespace-mtls-d025c-cust-5fd9effa",
			expectedURL:      "https://5a464-another-service-custom-namespace-mtls-d025c-custom-domain-443.internal",
			expectedUpstream: "https://another-service.custom-namespace-custom.domain:443",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result, err := buildAgentEndpoint(tc.irVirtualHost, tc.irService, tc.clusterDomain, tc.metadata)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedName, result.Name, "unexpected name for test case: %s", tc.name)
			assert.Equal(t, tc.irService.Namespace, result.Namespace, "unexpected namespace for test case: %s", tc.name)
			assert.Equal(t, tc.irVirtualHost.LabelsToAdd, result.Labels, "unexpected labels for test case: %s", tc.name)
			assert.Equal(t, tc.irVirtualHost.AnnotationsToAdd, result.Annotations, "unexpected annotations for test case: %s", tc.name)
			assert.Equal(t, tc.metadata, result.Spec.Metadata, "unexpected metadata for test case: %s", tc.name)
			assert.Equal(t, tc.expectedURL, result.Spec.URL, "unexpected URL for test case: %s", tc.name)
			assert.Equal(t, tc.expectedUpstream, result.Spec.Upstream.URL, "unexpected upstream URL for test case: %s", tc.name)
			assert.ElementsMatch(t, []string{}, result.Spec.Bindings, "unexpected bindings for test case: %s", tc.name)
			assert.Equal(t, tc.irService.Protocol, result.Spec.Upstream.Protocol, "unexpected upstream protocol for test case: %s", tc.name)
		})
	}
}

func TestBuildCloudEndpoint(t *testing.T) {
	testCases := []struct {
		testName     string
		irVHost      *ir.IRVirtualHost
		expectedName string
	}{
		{
			testName: "Basic setup",
			irVHost: &ir.IRVirtualHost{
				Bindings:  []string{"public"},
				Namespace: "default",
				Metadata:  "test-metadata",
				Listener: ir.IRListener{
					Hostname: "cloud-host",
					Port:     80,
					Protocol: ir.IRProtocol_HTTP,
				},
			},
			expectedName: "cloud-host",
		},
		{
			testName: "Custom namespace, annotations, and labels",
			irVHost: &ir.IRVirtualHost{
				Bindings:         []string{"public"},
				Namespace:        "foo",
				Metadata:         "test-metadata",
				LabelsToAdd:      map[string]string{"test-label": "test-label-val"},
				AnnotationsToAdd: map[string]string{"test-annotations": "test-annotation-val"},
				Listener: ir.IRListener{
					Hostname: "cloud-host",
					Port:     443,
					Protocol: ir.IRProtocol_HTTPS,
				},
			},
			expectedName: "cloud-host",
		},
		{
			testName: "Pooling enabled",
			irVHost: &ir.IRVirtualHost{
				Bindings:               []string{"public"},
				Namespace:              "foo",
				Metadata:               "test-metadata",
				EndpointPoolingEnabled: true,
				LabelsToAdd:            map[string]string{"test-label": "test-label-val"},
				AnnotationsToAdd:       map[string]string{"test-annotations": "test-annotation-val"},
				Listener: ir.IRListener{
					Hostname: "cloud-host",
					Port:     443,
					Protocol: ir.IRProtocol_HTTPS,
				},
			},
			expectedName: "cloud-host",
		},
		{
			testName: "Name prefix",
			irVHost: &ir.IRVirtualHost{
				Bindings:               []string{"public"},
				NamePrefix:             stringPtr("prefix"),
				Namespace:              "foo",
				Metadata:               "test-metadata",
				EndpointPoolingEnabled: true,
				LabelsToAdd:            map[string]string{"test-label": "test-label-val"},
				AnnotationsToAdd:       map[string]string{"test-annotations": "test-annotation-val"},
				Listener: ir.IRListener{
					Hostname: "cloud-host",
					Port:     443,
					Protocol: ir.IRProtocol_HTTPS,
				},
			},
			expectedName: "prefix-cloud-host",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			result, err := buildCloudEndpoint(tc.irVHost)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedName, result.Name, "unexpected name for test case")
			assert.Equal(t, tc.irVHost.Namespace, result.Namespace, "unexpected namespace for test case")
			assert.Equal(t, tc.irVHost.LabelsToAdd, result.Labels, "unexpected labels for test case")
			assert.Equal(t, tc.irVHost.AnnotationsToAdd, result.Annotations, "unexpected annotations for test case")
			assert.Equal(t, tc.irVHost.Metadata, result.Spec.Metadata, "unexpected metadata for test case")
		})
	}
}

func TestBuildDefaultDestinationPolicy(t *testing.T) {
	testCases := []struct {
		name               string
		irVHost            *ir.IRVirtualHost
		childEndpointCache map[ir.IRServiceKey]*ngrokv1alpha1.AgentEndpoint
		expectedPolicy     *trafficpolicy.TrafficPolicy
		expectedCacheKeys  []ir.IRServiceKey
	}{
		{
			name: "No default destination",
			irVHost: &ir.IRVirtualHost{
				DefaultDestination: nil,
			},
			childEndpointCache: map[ir.IRServiceKey]*ngrokv1alpha1.AgentEndpoint{},
			expectedPolicy:     trafficpolicy.NewTrafficPolicy(),
			expectedCacheKeys:  []ir.IRServiceKey{},
		},
		{
			name: "Default destination has a traffic policy",
			irVHost: &ir.IRVirtualHost{
				DefaultDestination: &ir.IRDestination{
					TrafficPolicies: []*trafficpolicy.TrafficPolicy{
						{
							OnHTTPRequest: []trafficpolicy.Rule{
								{
									Name: "Fallback 404 Rule",
									Actions: []trafficpolicy.Action{
										{
											Type: "custom-response",
											Config: map[string]interface{}{
												"content": "Fallback 404 page",
												"headers": map[string]string{
													"content-type": "text/plain",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			childEndpointCache: map[ir.IRServiceKey]*ngrokv1alpha1.AgentEndpoint{},
			expectedPolicy: &trafficpolicy.TrafficPolicy{
				OnHTTPRequest: []trafficpolicy.Rule{
					{
						Name: "Fallback 404 Rule",
						Actions: []trafficpolicy.Action{
							{
								Type: "custom-response",
								Config: map[string]interface{}{
									"content": "Fallback 404 page",
									"headers": map[string]string{
										"content-type": "text/plain",
									},
								},
							},
						},
					},
				},
				OnHTTPResponse: []trafficpolicy.Rule{},
				OnTCPConnect:   []trafficpolicy.Rule{},
			},
			expectedCacheKeys: []ir.IRServiceKey{},
		},
		{
			name: "Default destination has an upstream service",
			irVHost: &ir.IRVirtualHost{
				Listener: ir.IRListener{
					Hostname: "cloud-host",
					Port:     443,
					Protocol: ir.IRProtocol_HTTPS,
				},
				DefaultDestination: &ir.IRDestination{
					Upstream: &ir.IRUpstream{
						Service: ir.IRService{
							UID:       "service-uid",
							Namespace: "default",
							Name:      "test-service",
							Port:      8080,
							Scheme:    ir.IRScheme_HTTP,
						},
					},
				},
				LabelsToAdd:      map[string]string{"label": "value"},
				AnnotationsToAdd: map[string]string{"anno": "val"},
			},
			childEndpointCache: map[ir.IRServiceKey]*ngrokv1alpha1.AgentEndpoint{},
			expectedPolicy: &trafficpolicy.TrafficPolicy{
				OnHTTPRequest: []trafficpolicy.Rule{
					{
						Name: "Generated-Route-Default-Backend",
						Actions: []trafficpolicy.Action{
							{
								Type: "forward-internal",
								Config: map[string]interface{}{
									"url": "https://62d2f-test-service-default-cluster-local-8080.internal",
								},
							},
						},
					},
				},
				OnHTTPResponse: []trafficpolicy.Rule{},
				OnTCPConnect:   []trafficpolicy.Rule{},
			},
			expectedCacheKeys: []ir.IRServiceKey{
				ir.IRServiceKey("service-uid/default/test-service/8080"),
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			translator := &translator{
				clusterDomain:          "cluster.local",
				managedResourceLabels:  map[string]string{"app": "test"},
				defaultIngressMetadata: "test-metadata",
			}

			resultPolicy, err := translator.buildDefaultDestinationPolicy(tc.irVHost, tc.childEndpointCache)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedPolicy, resultPolicy, "unexpected policy for test case: %s", tc.name)

			var cacheKeys []ir.IRServiceKey
			for key := range tc.childEndpointCache {
				cacheKeys = append(cacheKeys, key)
			}
			assert.ElementsMatch(t, tc.expectedCacheKeys, cacheKeys, "unexpected cache keys for test case: %s", tc.name)
		})
	}
}

func TestGatewayMethodToIR(t *testing.T) {
	methodPtr := func(v gatewayv1.HTTPMethod) *gatewayv1.HTTPMethod {
		return &v
	}

	irPtr := func(v ir.IRMethodMatch) *ir.IRMethodMatch {
		return &v
	}

	testCases := []struct {
		name     string
		input    *gatewayv1.HTTPMethod
		expected *ir.IRMethodMatch
	}{
		{
			name:     "nil input returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "GET",
			input:    methodPtr(gatewayv1.HTTPMethodGet),
			expected: irPtr(ir.IRMethodMatch_Get),
		},
		{
			name:     "HEAD",
			input:    methodPtr(gatewayv1.HTTPMethodHead),
			expected: irPtr(ir.IRMethodMatch_Head),
		},
		{
			name:     "POST",
			input:    methodPtr(gatewayv1.HTTPMethodPost),
			expected: irPtr(ir.IRMethodMatch_Post),
		},
		{
			name:     "PUT",
			input:    methodPtr(gatewayv1.HTTPMethodPut),
			expected: irPtr(ir.IRMethodMatch_Put),
		},
		{
			name:     "DELETE",
			input:    methodPtr(gatewayv1.HTTPMethodDelete),
			expected: irPtr(ir.IRMethodMatch_Delete),
		},
		{
			name:     "CONNECT",
			input:    methodPtr(gatewayv1.HTTPMethodConnect),
			expected: irPtr(ir.IRMethodMatch_Connect),
		},
		{
			name:     "OPTIONS",
			input:    methodPtr(gatewayv1.HTTPMethodOptions),
			expected: irPtr(ir.IRMethodMatch_Options),
		},
		{
			name:     "TRACE",
			input:    methodPtr(gatewayv1.HTTPMethodTrace),
			expected: irPtr(ir.IRMethodMatch_Trace),
		},
		{
			name:     "PATCH",
			input:    methodPtr(gatewayv1.HTTPMethodPatch),
			expected: irPtr(ir.IRMethodMatch_Patch),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := gatewayMethodToIR(tc.input)
			if tc.expected == nil {
				assert.Nil(t, result, "expected nil for test case: %s", tc.name)
			} else {
				assert.NotNil(t, result, "expected non-nil for test case: %s", tc.name)
				assert.Equal(t, *tc.expected, *result, "unexpected conversion for test case: %s", tc.name)
			}
		})
	}
}

// TranslatorRawTestCase facuilitates the initial loading of test input/expected objects, but k8s objects with embedded structs don't parse cleanly
// with regular yaml marshalling so we need to be a little creative about how we process them.
type TranslatorRawTestCase struct {
	Input struct {
		GatewayClasses  []map[string]interface{} `yaml:"gatewayClasses"`
		Gateways        []map[string]interface{} `yaml:"gateways"`
		HTTPRoutes      []map[string]interface{} `yaml:"httpRoutes"`
		TCPRoutes       []map[string]interface{} `yaml:"tcpRoutes"`
		TLSRoutes       []map[string]interface{} `yaml:"tlsRoutes"`
		IngressClasses  []map[string]interface{} `yaml:"ingressClasses"`
		Ingresses       []map[string]interface{} `yaml:"ingresses"`
		TrafficPolicies []map[string]interface{} `yaml:"trafficPolicies"`
		Services        []map[string]interface{} `yaml:"services"`
		Secrets         []map[string]interface{} `yaml:"secrets"`
		Configmaps      []map[string]interface{} `yaml:"configMaps"`
		Namespaces      []map[string]interface{} `yaml:"namespaces"`
		ReferenceGrants []map[string]interface{} `yaml:"referenceGrants"`
	} `yaml:"input"`

	Expected struct {
		CloudEndpoints []map[string]interface{} `yaml:"cloudEndpoints"`
		AgentEndpoints []map[string]interface{} `yaml:"agentEndpoints"`
	} `yaml:"expected"`
}

// TranslatorTestCase stores our actual fully parsed inputs/outputs
type TranslatorTestCase struct {
	Input struct {
		GatewayClasses  []*gatewayv1.GatewayClass
		Gateways        []*gatewayv1.Gateway
		HTTPRoutes      []*gatewayv1.HTTPRoute
		TCPRoutes       []*gatewayv1alpha2.TCPRoute
		TLSRoutes       []*gatewayv1alpha2.TLSRoute
		IngressClasses  []*netv1.IngressClass
		Ingresses       []*netv1.Ingress
		TrafficPolicies []*ngrokv1alpha1.NgrokTrafficPolicy
		Secrets         []*corev1.Secret
		ConfigMaps      []*corev1.ConfigMap
		Services        []*corev1.Service
		Namespaces      []*corev1.Namespace
		ReferenceGrants []*gatewayv1beta1.ReferenceGrant
	}

	Expected struct {
		CloudEndpoints []*ngrokv1alpha1.CloudEndpoint
		AgentEndpoints []*ngrokv1alpha1.AgentEndpoint
	}
}

func toIDs[T client.Object](objs []T) []string {
	items := []string{}
	for _, obj := range objs {
		items = append(items, fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName()))
	}
	return items
}

func TestTranslate(t *testing.T) {
	testdataDir := "testdata/translator"
	disableRefGrantsDir := "testdata/translator-disable-refgrants"

	// Create a scheme with all supported types
	sch := runtime.NewScheme()

	utilruntime.Must(gatewayv1.Install(sch))
	utilruntime.Must(gatewayv1beta1.Install(sch))
	utilruntime.Must(gatewayv1alpha2.Install(sch))
	utilruntime.Must(clientgoscheme.AddToScheme(sch))
	utilruntime.Must(ingressv1alpha1.AddToScheme(sch))
	utilruntime.Must(corev1.AddToScheme(sch))
	utilruntime.Must(ngrokv1alpha1.AddToScheme(sch))

	// Load test files from the testdata directory
	defaultTranslatorFiles, err := filepath.Glob(filepath.Join(testdataDir, "*.yaml"))
	require.NoError(t, err, "failed to read test files in %s", testdataDir)
	disableRefGrantsFiles, err := filepath.Glob(filepath.Join(disableRefGrantsDir, "*.yaml"))
	require.NoError(t, err, "failed to read test files in %s", disableRefGrantsDir)

	for _, file := range defaultTranslatorFiles {
		logger := logr.New(logr.Discard().GetSink())
		// If you need to debug tests, uncomment this logger instead to actually see errors printed in the tests.
		// Otherwise, keep the above logger so that we don't output stuff and make the test output harder to read.
		// logger = testr.New(t)

		driver := NewDriver(
			logger,
			sch,
			testutils.DefaultControllerName,
			types.NamespacedName{
				Name:      "test-manager-name",
				Namespace: "test-manager-namespace",
			},
			WithGatewayEnabled(true),
			WithSyncAllowConcurrent(true),
			WithGatewayTCPRouteEnabled(true),
			WithGatewayTLSRouteEnabled(true),
		)
		t.Run(filepath.Base(file), func(t *testing.T) {
			tc := loadTranslatorTestCase(t, file, sch)

			// Load input objects into the driver store
			inputObjects := loadTranslatorInputObjs(t, tc)
			client := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(inputObjects...).Build()

			require.NoError(t, driver.Seed(t.Context(), client))
			translator := NewTranslator(
				driver.log,
				driver.store,
				driver.defaultManagedResourceLabels(),
				driver.ingressNgrokMetadata,
				driver.gatewayNgrokMetadata,
				"svc.cluster.local",
				false, // Require reference grants (default)
			)

			// Finally, run translate and check the contents
			result := translator.Translate()
			assert.Equal(t, len(tc.Expected.CloudEndpoints), len(result.CloudEndpoints), fmt.Sprintf("mismatch in actual vs expected number of cloud endpoints from translation, expected: %v, actual: %v", toIDs(tc.Expected.CloudEndpoints), result.CloudEndpoints))
			assert.Equal(t, len(tc.Expected.AgentEndpoints), len(result.AgentEndpoints), fmt.Sprintf("mismatch in actual vs expected number of agent endpoints from translation, expected: %v, actual: %v", toIDs(tc.Expected.AgentEndpoints), result.AgentEndpoints))

			for _, expectedCLEP := range tc.Expected.CloudEndpoints {
				actualCLEP, exists := result.CloudEndpoints[types.NamespacedName{
					Name:      expectedCLEP.Name,
					Namespace: expectedCLEP.Namespace,
				}]
				require.True(t, exists, "expected CloudEndpoint %s.%s to exist, content: %v", expectedCLEP.Name, expectedCLEP.Namespace, result.CloudEndpoints)
				assert.Equal(t, expectedCLEP.Name, actualCLEP.Name)
				assert.Equal(t, expectedCLEP.Namespace, actualCLEP.Namespace)
				assert.Equal(t, expectedCLEP.Labels, actualCLEP.Labels)
				assert.Equal(t, expectedCLEP.Annotations, actualCLEP.Annotations)
				assert.Equal(t, expectedCLEP.Spec.URL, actualCLEP.Spec.URL)
				assert.Equal(t, expectedCLEP.Spec.TrafficPolicyName, actualCLEP.Spec.TrafficPolicyName)
				assert.Equal(t, expectedCLEP.Spec.PoolingEnabled, actualCLEP.Spec.PoolingEnabled)
				if expectedCLEP.Spec.TrafficPolicy != nil {
					require.NotNil(t, actualCLEP.Spec.TrafficPolicy)
					expectedTrafficPolicyCfg := &trafficpolicy.TrafficPolicy{}
					require.NoError(t, json.Unmarshal(expectedCLEP.Spec.TrafficPolicy.Policy, expectedTrafficPolicyCfg))

					actualTrafficPolicyCfg := &trafficpolicy.TrafficPolicy{}
					require.NoError(t, json.Unmarshal(actualCLEP.Spec.TrafficPolicy.Policy, actualTrafficPolicyCfg))
					assert.Equal(t, expectedTrafficPolicyCfg, actualTrafficPolicyCfg)
				} else {
					assert.Nil(t, actualCLEP.Spec.TrafficPolicy)
				}
				assert.Equal(t, expectedCLEP.Spec.Description, actualCLEP.Spec.Description)
				assert.Equal(t, expectedCLEP.Spec.Metadata, actualCLEP.Spec.Metadata)
				assert.Equal(t, expectedCLEP.Spec.Bindings, actualCLEP.Spec.Bindings)
			}

			for _, expectedAE := range tc.Expected.AgentEndpoints {
				actualAE, exists := result.AgentEndpoints[types.NamespacedName{
					Name:      expectedAE.Name,
					Namespace: expectedAE.Namespace,
				}]
				require.True(t, exists, "expected AgentEndpoint %s.%s to exist. actual agent endpoints: %v", expectedAE.Name, expectedAE.Namespace, result.AgentEndpoints)
				assert.Equal(t, expectedAE.Name, actualAE.Name)
				assert.Equal(t, expectedAE.Namespace, actualAE.Namespace)
				assert.Equal(t, expectedAE.Labels, actualAE.Labels)
				assert.Equal(t, expectedAE.Annotations, actualAE.Annotations)
				assert.Equal(t, expectedAE.Spec.URL, actualAE.Spec.URL)
				if expectedAE.Spec.TrafficPolicy != nil {
					require.NotNil(t, actualAE.Spec.TrafficPolicy)
					assert.Equal(t, expectedAE.Spec.TrafficPolicy.Reference, actualAE.Spec.TrafficPolicy.Reference)

					expectedTrafficPolicyCfg := &trafficpolicy.TrafficPolicy{}
					require.NoError(t, json.Unmarshal(expectedAE.Spec.TrafficPolicy.Inline, expectedTrafficPolicyCfg))

					actualTrafficPolicyCfg := &trafficpolicy.TrafficPolicy{}
					require.NoError(t, json.Unmarshal(actualAE.Spec.TrafficPolicy.Inline, actualTrafficPolicyCfg))
					assert.Equal(t, expectedTrafficPolicyCfg, actualTrafficPolicyCfg)
				} else {
					assert.Nil(t, actualAE.Spec.TrafficPolicy)
				}
				assert.Equal(t, expectedAE.Spec.Description, actualAE.Spec.Description)
				assert.Equal(t, expectedAE.Spec.Metadata, actualAE.Spec.Metadata)
				assert.Equal(t, expectedAE.Spec.Bindings, actualAE.Spec.Bindings)
				assert.Equal(t, expectedAE.Spec.Upstream.Protocol, actualAE.Spec.Upstream.Protocol)
				assert.Equal(t, expectedAE.Spec.Upstream.URL, actualAE.Spec.Upstream.URL)
				assert.Equal(t, expectedAE.Spec.Upstream.ProxyProtocolVersion, actualAE.Spec.Upstream.ProxyProtocolVersion)
			}

		})
	}
	for _, file := range disableRefGrantsFiles {
		logger := logr.New(logr.Discard().GetSink())
		// If you need to debug tests, uncomment this logger instead to actually see errors printed in the tests.
		// Otherwise, keep the above logger so that we don't output stuff and make the test output harder to read.
		// logger = testr.New(t)

		driver := NewDriver(
			logger,
			sch,
			testutils.DefaultControllerName,
			types.NamespacedName{
				Name:      "test-manager-name",
				Namespace: "test-manager-namespace",
			},
			WithGatewayEnabled(true),
			WithSyncAllowConcurrent(true),
		)
		t.Run(filepath.Base(file), func(t *testing.T) {
			tc := loadTranslatorTestCase(t, file, sch)

			// Load input objects into the driver store
			inputObjects := loadTranslatorInputObjs(t, tc)
			client := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(inputObjects...).Build()

			require.NoError(t, driver.Seed(t.Context(), client))
			translator := NewTranslator(
				driver.log,
				driver.store,
				driver.defaultManagedResourceLabels(),
				driver.ingressNgrokMetadata,
				driver.gatewayNgrokMetadata,
				"svc.cluster.local",
				true, // Disable reference grants
			)

			// Finally, run translate and check the contents
			result := translator.Translate()
			require.Equal(t, len(tc.Expected.AgentEndpoints), len(result.AgentEndpoints))
			require.Equal(t, len(tc.Expected.CloudEndpoints), len(result.CloudEndpoints))

			for _, expectedCLEP := range tc.Expected.CloudEndpoints {
				actualCLEP, exists := result.CloudEndpoints[types.NamespacedName{
					Name:      expectedCLEP.Name,
					Namespace: expectedCLEP.Namespace,
				}]
				require.True(t, exists, "expected CloudEndpoint %s.%s to exist, content: %v", expectedCLEP.Name, expectedCLEP.Namespace, result.CloudEndpoints)
				assert.Equal(t, expectedCLEP.Name, actualCLEP.Name)
				assert.Equal(t, expectedCLEP.Namespace, actualCLEP.Namespace)
				assert.Equal(t, expectedCLEP.Labels, actualCLEP.Labels)
				assert.Equal(t, expectedCLEP.Annotations, actualCLEP.Annotations)
				assert.Equal(t, expectedCLEP.Spec.URL, actualCLEP.Spec.URL)
				assert.Equal(t, expectedCLEP.Spec.TrafficPolicyName, actualCLEP.Spec.TrafficPolicyName)
				assert.Equal(t, expectedCLEP.Spec.PoolingEnabled, actualCLEP.Spec.PoolingEnabled)
				if expectedCLEP.Spec.TrafficPolicy != nil {
					require.NotNil(t, actualCLEP.Spec.TrafficPolicy)
					expectedTrafficPolicyCfg := &trafficpolicy.TrafficPolicy{}
					require.NoError(t, json.Unmarshal(expectedCLEP.Spec.TrafficPolicy.Policy, expectedTrafficPolicyCfg))

					actualTrafficPolicyCfg := &trafficpolicy.TrafficPolicy{}
					require.NoError(t, json.Unmarshal(actualCLEP.Spec.TrafficPolicy.Policy, actualTrafficPolicyCfg))
					assert.Equal(t, expectedTrafficPolicyCfg, actualTrafficPolicyCfg)
				} else {
					assert.Nil(t, actualCLEP.Spec.TrafficPolicy)
				}
				assert.Equal(t, expectedCLEP.Spec.Description, actualCLEP.Spec.Description)
				assert.Equal(t, expectedCLEP.Spec.Metadata, actualCLEP.Spec.Metadata)
				assert.Equal(t, expectedCLEP.Spec.Bindings, actualCLEP.Spec.Bindings)
			}

			for _, expectedAE := range tc.Expected.AgentEndpoints {
				actualAE, exists := result.AgentEndpoints[types.NamespacedName{
					Name:      expectedAE.Name,
					Namespace: expectedAE.Namespace,
				}]
				require.True(t, exists, "expected AgentEndpoint %s.%s to exist. actual agent endpoints: %v", expectedAE.Name, expectedAE.Namespace, result.AgentEndpoints)
				assert.Equal(t, expectedAE.Name, actualAE.Name)
				assert.Equal(t, expectedAE.Namespace, actualAE.Namespace)
				assert.Equal(t, expectedAE.Labels, actualAE.Labels)
				assert.Equal(t, expectedAE.Annotations, actualAE.Annotations)
				assert.Equal(t, expectedAE.Spec.URL, actualAE.Spec.URL)
				if expectedAE.Spec.TrafficPolicy != nil {
					require.NotNil(t, actualAE.Spec.TrafficPolicy)
					assert.Equal(t, expectedAE.Spec.TrafficPolicy.Reference, actualAE.Spec.TrafficPolicy.Reference)

					expectedTrafficPolicyCfg := &trafficpolicy.TrafficPolicy{}
					require.NoError(t, json.Unmarshal(expectedAE.Spec.TrafficPolicy.Inline, expectedTrafficPolicyCfg))

					actualTrafficPolicyCfg := &trafficpolicy.TrafficPolicy{}
					require.NoError(t, json.Unmarshal(actualAE.Spec.TrafficPolicy.Inline, actualTrafficPolicyCfg))
					assert.Equal(t, expectedTrafficPolicyCfg, actualTrafficPolicyCfg)
				} else {
					assert.Nil(t, actualAE.Spec.TrafficPolicy)
				}
				assert.Equal(t, expectedAE.Spec.Description, actualAE.Spec.Description)
				assert.Equal(t, expectedAE.Spec.Metadata, actualAE.Spec.Metadata)
				assert.Equal(t, expectedAE.Spec.Bindings, actualAE.Spec.Bindings)
				assert.Equal(t, expectedAE.Spec.Upstream.Protocol, actualAE.Spec.Upstream.Protocol)
				assert.Equal(t, expectedAE.Spec.Upstream.URL, actualAE.Spec.Upstream.URL)
				assert.Equal(t, expectedAE.Spec.Upstream.ProxyProtocolVersion, actualAE.Spec.Upstream.ProxyProtocolVersion)
			}

		})
	}
}

func loadTranslatorInputObjs(t *testing.T, tc TranslatorTestCase) []runtime.Object {
	t.Helper()
	inputObjects := []runtime.Object{}
	for _, obj := range tc.Input.GatewayClasses {
		inputObjects = append(inputObjects, obj)
	}
	for _, obj := range tc.Input.Gateways {
		inputObjects = append(inputObjects, obj)
	}
	for _, obj := range tc.Input.HTTPRoutes {
		inputObjects = append(inputObjects, obj)
	}
	for _, obj := range tc.Input.TLSRoutes {
		inputObjects = append(inputObjects, obj)
	}
	for _, obj := range tc.Input.TCPRoutes {
		inputObjects = append(inputObjects, obj)
	}
	for _, obj := range tc.Input.ReferenceGrants {
		inputObjects = append(inputObjects, obj)
	}
	for _, obj := range tc.Input.IngressClasses {
		inputObjects = append(inputObjects, obj)
	}
	for _, obj := range tc.Input.Ingresses {
		inputObjects = append(inputObjects, obj)
	}
	for _, obj := range tc.Input.TrafficPolicies {
		inputObjects = append(inputObjects, obj)
	}
	for _, obj := range tc.Input.Services {
		inputObjects = append(inputObjects, obj)
	}
	for _, obj := range tc.Input.Secrets {
		inputObjects = append(inputObjects, obj)
	}
	for _, obj := range tc.Input.ConfigMaps {
		inputObjects = append(inputObjects, obj)
	}
	for _, obj := range tc.Input.Namespaces {
		inputObjects = append(inputObjects, obj)
	}
	return inputObjects
}

func loadTranslatorTestCase(t *testing.T, file string, sch *runtime.Scheme) TranslatorTestCase {
	t.Helper()
	data, err := os.ReadFile(file)
	require.NoError(t, err, "failed to read file: %s", file)

	// Load into the RawTestCase
	rawTC := new(TranslatorRawTestCase)
	require.NoError(t, yaml.UnmarshalStrict(data, rawTC), "failed to unmarshal raw testCase")

	// Use scheme based decoding to properly parse everything into TestCase
	tc := TranslatorTestCase{}

	// Decode input objects
	for _, rawObj := range rawTC.Input.GatewayClasses {

		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		gatewayClass, ok := obj.(*gatewayv1.GatewayClass)
		require.True(t, ok, "expected a GatewayClass, got %T", obj)
		tc.Input.GatewayClasses = append(tc.Input.GatewayClasses, gatewayClass)
	}
	for _, rawObj := range rawTC.Input.Gateways {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		gateway, ok := obj.(*gatewayv1.Gateway)
		require.True(t, ok, "expected a Gateway, got %T", obj)
		tc.Input.Gateways = append(tc.Input.Gateways, gateway)
	}
	for _, rawObj := range rawTC.Input.HTTPRoutes {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		httpRoute, ok := obj.(*gatewayv1.HTTPRoute)
		require.True(t, ok, "expected an HTTPRoute, got %T", obj)
		tc.Input.HTTPRoutes = append(tc.Input.HTTPRoutes, httpRoute)
	}
	for _, rawObj := range rawTC.Input.TCPRoutes {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		tcpRoute, ok := obj.(*gatewayv1alpha2.TCPRoute)
		require.True(t, ok, "expected a TCPRoute, got %T", obj)
		tc.Input.TCPRoutes = append(tc.Input.TCPRoutes, tcpRoute)
	}
	for _, rawObj := range rawTC.Input.TLSRoutes {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		tlsRoute, ok := obj.(*gatewayv1alpha2.TLSRoute)
		require.True(t, ok, "expected a TLSRoute, got %T", obj)
		tc.Input.TLSRoutes = append(tc.Input.TLSRoutes, tlsRoute)
	}
	for _, rawObj := range rawTC.Input.ReferenceGrants {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		referenceGrant, ok := obj.(*gatewayv1beta1.ReferenceGrant)
		require.True(t, ok, "expected a ReferenceGrant, got %T", obj)
		tc.Input.ReferenceGrants = append(tc.Input.ReferenceGrants, referenceGrant)
	}
	for _, rawObj := range rawTC.Input.IngressClasses {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		ingClass, ok := obj.(*netv1.IngressClass)
		require.True(t, ok, "expected an IngressClass, got %T", obj)
		tc.Input.IngressClasses = append(tc.Input.IngressClasses, ingClass)
	}
	for _, rawObj := range rawTC.Input.Ingresses {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		ing, ok := obj.(*netv1.Ingress)
		require.True(t, ok, "expected an Ingress, got %T", obj)
		tc.Input.Ingresses = append(tc.Input.Ingresses, ing)
	}
	for _, rawObj := range rawTC.Input.Services {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		svc, ok := obj.(*corev1.Service)
		require.True(t, ok, "expected a Service, got %T", obj)
		tc.Input.Services = append(tc.Input.Services, svc)
	}
	for _, rawObj := range rawTC.Input.Secrets {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		secret, ok := obj.(*corev1.Secret)
		require.True(t, ok, "expected a Secret, got %T", obj)
		tc.Input.Secrets = append(tc.Input.Secrets, secret)
	}
	for _, rawObj := range rawTC.Input.Configmaps {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		configMap, ok := obj.(*corev1.ConfigMap)
		require.True(t, ok, "expected a ConfigMap, got %T", obj)
		tc.Input.ConfigMaps = append(tc.Input.ConfigMaps, configMap)
	}
	for _, rawObj := range rawTC.Input.Namespaces {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		namespace, ok := obj.(*corev1.Namespace)
		require.True(t, ok, "expected a Namespace, got %T", obj)
		tc.Input.Namespaces = append(tc.Input.Namespaces, namespace)
	}
	for _, rawObj := range rawTC.Input.TrafficPolicies {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		pol, ok := obj.(*ngrokv1alpha1.NgrokTrafficPolicy)
		require.True(t, ok, "expected an NgrokTrafficPolicy, got %T", obj)
		tc.Input.TrafficPolicies = append(tc.Input.TrafficPolicies, pol)
	}

	// Decode expected objects
	for _, rawObj := range rawTC.Expected.CloudEndpoints {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		ce, ok := obj.(*ngrokv1alpha1.CloudEndpoint)
		require.True(t, ok, "expected a CloudEndpoint, got %T", obj)
		tc.Expected.CloudEndpoints = append(tc.Expected.CloudEndpoints, ce)
	}
	for _, rawObj := range rawTC.Expected.AgentEndpoints {
		obj, err := decodeViaScheme(sch, rawObj)
		require.NoError(t, err)
		ae, ok := obj.(*ngrokv1alpha1.AgentEndpoint)
		require.True(t, ok, "expected an AgentEndpoint, got %T", obj)
		tc.Expected.AgentEndpoints = append(tc.Expected.AgentEndpoints, ae)
	}
	return tc
}

// decodeViaScheme helps us decode raw objects loaded from test data yaml files into proper objects that can then be typecast
func decodeViaScheme(s *runtime.Scheme, rawObj map[string]interface{}) (runtime.Object, error) {
	// Convert map to YAML
	y, err := yaml.Marshal(rawObj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw map to YAML: %w", err)
	}

	// Decode
	decoder := serializer.NewCodecFactory(s).UniversalDeserializer()
	obj, _, err := decoder.Decode(y, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode via scheme: %w", err)
	}

	if obj != nil {
		// Clear kind for testing
		kind := obj.GetObjectKind()
		kind.SetGroupVersionKind(schema.GroupVersionKind{})
	}

	return obj, nil
}
