// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package extensions

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// mockSTS returns mock Azure AD responses so tests don't have to account for MSAL metadata requests
type mockSTS struct {
	tokenRequestCallback func(*http.Request)
}

func (m *mockSTS) Do(req *http.Request) (*http.Response, error) {
	res := http.Response{StatusCode: http.StatusOK}
	switch s := strings.Split(req.URL.Path, "/"); s[len(s)-1] {
	case "instance":
		res.Body = io.NopCloser(bytes.NewReader(instanceMetadata("tenant")))
	case "openid-configuration":
		res.Body = io.NopCloser(bytes.NewReader(tenantMetadata("tenant")))
	case "token":
		if m.tokenRequestCallback != nil {
			m.tokenRequestCallback(req)
		}
		if err := req.ParseForm(); err != nil {
			return nil, err
		}
		scope := strings.Split(req.FormValue("scope"), " ")[0]
		userinfo := ""
		if upn := req.FormValue("username"); upn != "" {
			clientinfo := base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprintf(`{"uid":"%s","utid":"utid"}`, upn)))
			userinfo = fmt.Sprintf(`, "client_info":"%s", "id_token":"x.e30", "refresh_token": "rt"`, clientinfo)
		}
		res.Body = io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"access_token": %q, "expires_in": 3600%s}`, scope, userinfo))))
	default:
		// User realm metadata request paths look like "/common/UserRealm/user@domain".
		// Matching on the UserRealm segment avoids having to know the UPN.
		if s[len(s)-2] == "UserRealm" {
			res.Body = io.NopCloser(
				strings.NewReader(`{"account_type":"Managed","cloud_audience_urn":"urn","cloud_instance_name":"...","domain_name":"..."}`),
			)
		} else {
			panic("unexpected request " + req.URL.String())
		}
	}
	return &res, nil
}

func (m *mockSTS) CloseIdleConnections() {}

func instanceMetadata(tenant string) []byte {
	return []byte(strings.ReplaceAll(`{
		"tenant_discovery_endpoint": "https://login.microsoftonline.com/{tenant}/v2.0/.well-known/openid-configuration",
		"api-version": "1.1",
		"metadata": [
			{
				"preferred_network": "login.microsoftonline.com",
				"preferred_cache": "login.windows.net",
				"aliases": [
					"login.microsoftonline.com",
					"login.windows.net",
					"login.microsoft.com",
					"sts.windows.net"
				]
			}
		]
	}`, "{tenant}", tenant))
}

func tenantMetadata(tenant string) []byte {
	return []byte(strings.ReplaceAll(`{
		"token_endpoint": "https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token",
		"token_endpoint_auth_methods_supported": [
			"client_secret_post",
			"private_key_jwt",
			"client_secret_basic"
		],
		"jwks_uri": "https://login.microsoftonline.com/{tenant}/discovery/v2.0/keys",
		"response_modes_supported": [
			"query",
			"fragment",
			"form_post"
		],
		"subject_types_supported": [
			"pairwise"
		],
		"id_token_signing_alg_values_supported": [
			"RS256"
		],
		"response_types_supported": [
			"code",
			"id_token",
			"code id_token",
			"id_token token"
		],
		"scopes_supported": [
			"openid",
			"profile",
			"email",
			"offline_access"
		],
		"issuer": "https://login.microsoftonline.com/{tenant}/v2.0",
		"request_uri_parameter_supported": false,
		"userinfo_endpoint": "https://graph.microsoft.com/oidc/userinfo",
		"authorization_endpoint": "https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize",
		"device_authorization_endpoint": "https://login.microsoftonline.com/{tenant}/oauth2/v2.0/devicecode",
		"http_logout_supported": true,
		"frontchannel_logout_supported": true,
		"end_session_endpoint": "https://login.microsoftonline.com/{tenant}/oauth2/v2.0/logout",
		"claims_supported": [
			"sub",
			"iss",
			"cloud_instance_name",
			"cloud_instance_host_name",
			"cloud_graph_host_name",
			"msgraph_host",
			"aud",
			"exp",
			"iat",
			"auth_time",
			"acr",
			"nonce",
			"preferred_username",
			"name",
			"tid",
			"ver",
			"at_hash",
			"c_hash",
			"email"
		],
		"kerberos_endpoint": "https://login.microsoftonline.com/{tenant}/kerberos",
		"tenant_region_scope": "NA",
		"cloud_instance_name": "microsoftonline.com",
		"cloud_graph_host_name": "graph.windows.net",
		"msgraph_host": "graph.microsoft.com",
		"rbac_url": "https://pas.windows.net"
	}`, "{tenant}", tenant))
}
