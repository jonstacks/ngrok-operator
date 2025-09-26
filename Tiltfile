load('ext://helm_resource', 'helm_resource', 'helm_repo')

# Output diagnostic messages
#   You can print log messages, warnings, and fatal errors, which will
#   appear in the (Tiltfile) resource in the web UI. Tiltfiles support
#   multiline strings and common string operations such as formatting.
#
#   More info: https://docs.tilt.dev/api.html#api.warn
print("""
-----------------------------------------------------------------
✨ Hello Tilt! This appears in the (Tiltfile) pane whenever Tilt
   evaluates this file.
-----------------------------------------------------------------
""".strip())


def ngrok_operator():
    helm_repo("ngrok", "https://charts.ngrok.com")
    helm_resource(
        "ngrok-operator",
        "ngrok/ngrok-operator",
        resource_deps=["ngrok"],
        namespace="ngrok-operator",
        flags=[
            "--create-namespace",
            "--set=credentials.apiKey={}".format(os.environ.get("NGROK_API_KEY", "")),
            "--set=credentials.authtoken={}".format(os.environ.get("NGROK_AUTHTOKEN", "")),
            "--set=defaultDomainReclaimPolicy=Retain",
            "--set=bindings.enabled=true",
        ],
    )

ngrok_operator()

k8s_yaml('deploy/domain.yaml')
k8s_resource(objects=['example-domain:Domain:default'], new_name='app', resource_deps=['ngrok-operator'])

if config.tilt_subcommand == 'down':
  local('kubectl delete domains.ingress.k8s.ngrok.com -A --all --wait --timeout=10s || true')
  local('kubectl delete boundendpoints -A --all --wait --timeout=10s || true')
  local('kubectl delete kubernetesoperators -A --all --wait --timeout=10s || true')
