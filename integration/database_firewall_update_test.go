package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite.Focus("database/firewalls", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	mockResponses := []*godo.DatabaseFirewallRule{
		{
			UUID:        "cdb689c2-56e6-48e6-869d-306c85af178d",
			ClusterUUID: "d168d635-1c88-4616-b9b4-793b7c573927",
			Type:        "ip_addr",
			Value:       "107.13.36.145",
			CreatedAt:   time.Now(),
		},
	}

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases/1/firewall":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				switch req.Method {
				case http.MethodGet:
					data, err := json.Marshal(map[string]interface{}{
						"rules": mockResponses,
					})
					if err != nil {
						t.Fatalf("%+v", err)
					}
					w.Write(data)

				case http.MethodPut:
					v := map[string][]*godo.DatabaseFirewallRule{
						"rules": make([]*godo.DatabaseFirewallRule, 0),
					}
					if err := json.NewDecoder(req.Body).Decode(&v); err != nil {
						t.Fatalf("%+v", err)
					}

					// We're assuming the PUT request will only include the type
					// and value, so we generate the UUID to make it more like the
					// actual implementation.
					rules, ok := v["rules"]
					if !ok {
						t.Fatalf("expected rules tp be present")
					}

					for _, rule := range rules {
						rule.UUID = "cdb089a2-56e6-48e6-869d-306c85af178d"
						rule.CreatedAt = time.Now()

						mockResponses = append(mockResponses, rule)
					}

					w.WriteHeader(http.StatusNoContent)
					return

				default:
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is update", func() {
		it("update a database cluster's firewall rules", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"firewalls",
				"update",
				"1",
				"--rules", "ip_addr:192.168.1.2",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))

			expected := strings.TrimSpace(databasesUpdateFirewallRuleOutput)
			actual := strings.TrimSpace(string(output))

			if expected != actual {
				t.Errorf("expected\n\n%s\n\nbut got\n\n%s\n\n", expected, actual)
			}
		})
	})

})

const (
	databasesUpdateFirewallRuleOutput = `UUID                                    ClusterUUID                             Type       Value            Created At
cdb689c2-56e6-48e6-869d-306c85af178d    d168d635-1c88-4616-b9b4-793b7c573927    ip_addr    107.13.36.145    2021-02-01 16:11:13.212099 -0500 EST
cdb089a2-56e6-48e6-869d-306c85af178d    1                                       ip_addr    192.168.1.2      2021-02-01 16:11:13.608441 -0500 EST
	`

	databasesUpdateFirewallRuleResponse = `{
		  "rules": [
			{
			  "uuid": "cdb689c2-56e6-48e6-869d-306c85af178d",
			  "cluster_uuid": "d168d635-1c88-4616-b9b4-793b7c573927",
			  "type": "ip_addr",
			  "value": "192.168.1.2",
			  "created_at": "2021-01-27T20:34:12Z"
			}
		  ]
		}`
)
