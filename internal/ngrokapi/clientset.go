package ngrokapi

import (
	"github.com/ngrok/ngrok-api-go/v5"
	tunnel_group_backends "github.com/ngrok/ngrok-api-go/v5/backends/tunnel_group"
	"github.com/ngrok/ngrok-api-go/v5/certificate_authorities"
	https_edges "github.com/ngrok/ngrok-api-go/v5/edges/https"
	https_edge_routes "github.com/ngrok/ngrok-api-go/v5/edges/https_routes"
	tcp_edges "github.com/ngrok/ngrok-api-go/v5/edges/tcp"
	"github.com/ngrok/ngrok-api-go/v5/ip_policies"
	"github.com/ngrok/ngrok-api-go/v5/ip_policy_rules"
	"github.com/ngrok/ngrok-api-go/v5/reserved_addrs"
	"github.com/ngrok/ngrok-api-go/v5/reserved_domains"
)

type Clientset interface {
	CertificateAuthorities() *certificate_authorities.Client
	Domains() *reserved_domains.Client
	EdgeModules() EdgeModulesClientset
	HTTPSEdges() *https_edges.Client
	HTTPSEdgeRoutes() *https_edge_routes.Client
	IPPolicies() *ip_policies.Client
	IPPolicyRules() *ip_policy_rules.Client
	TCPAddresses() *reserved_addrs.Client
	TCPEdges() *tcp_edges.Client
	TunnelGroupBackends() *tunnel_group_backends.Client
}

type DefaultClientset struct {
	certificateAuthoritiesClient *certificate_authorities.Client
	domainsClient                *reserved_domains.Client
	edgeModulesClientset         *defaultEdgeModulesClientset
	httpsEdgesClient             *https_edges.Client
	httpsEdgeRoutesClient        *https_edge_routes.Client
	ipPoliciesClient             *ip_policies.Client
	ipPolicyRulesClient          *ip_policy_rules.Client
	tcpAddrsClient               *reserved_addrs.Client
	tcpEdgesClient               *tcp_edges.Client
	tunnelGroupBackendsClient    *tunnel_group_backends.Client
}

// NewClientSet creates a new ClientSet from an ngrok client config.
func NewClientSet(config *ngrok.ClientConfig) *DefaultClientset {
	return &DefaultClientset{
		certificateAuthoritiesClient: certificate_authorities.NewClient(config),
		domainsClient:                reserved_domains.NewClient(config),
		edgeModulesClientset:         newEdgeModulesClientset(config),
		httpsEdgesClient:             https_edges.NewClient(config),
		httpsEdgeRoutesClient:        https_edge_routes.NewClient(config),
		ipPoliciesClient:             ip_policies.NewClient(config),
		ipPolicyRulesClient:          ip_policy_rules.NewClient(config),
		tcpAddrsClient:               reserved_addrs.NewClient(config),
		tcpEdgesClient:               tcp_edges.NewClient(config),
		tunnelGroupBackendsClient:    tunnel_group_backends.NewClient(config),
	}
}

func (c *DefaultClientset) CertificateAuthorities() *certificate_authorities.Client {
	return c.certificateAuthoritiesClient
}

func (c *DefaultClientset) Domains() *reserved_domains.Client {
	return c.domainsClient
}

func (c *DefaultClientset) EdgeModules() EdgeModulesClientset {
	return c.edgeModulesClientset
}

func (c *DefaultClientset) HTTPSEdges() *https_edges.Client {
	return c.httpsEdgesClient
}

func (c *DefaultClientset) HTTPSEdgeRoutes() *https_edge_routes.Client {
	return c.httpsEdgeRoutesClient
}

func (c *DefaultClientset) IPPolicies() *ip_policies.Client {
	return c.ipPoliciesClient
}

func (c *DefaultClientset) IPPolicyRules() *ip_policy_rules.Client {
	return c.ipPolicyRulesClient
}

func (c *DefaultClientset) TCPAddresses() *reserved_addrs.Client {
	return c.tcpAddrsClient
}

func (c *DefaultClientset) TCPEdges() *tcp_edges.Client {
	return c.tcpEdgesClient
}

func (c *DefaultClientset) TunnelGroupBackends() *tunnel_group_backends.Client {
	return c.tunnelGroupBackendsClient
}
