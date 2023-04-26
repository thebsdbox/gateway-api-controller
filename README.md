# gateway-api-controller

This is an initial implementation of the various controllers required in order to manage Gateway API network deployments within Kubernetes. At the moment four basic controllers are implemented:

- GatewayClass
- Gateway
- TCPRoute
- UDPRoute

## Usage

### Build

`go build` 

If I ever really learn how makefiles work, then perhaps i'll implement one

### Running

If you're running outside of a Kubernetes cluster then something like the following will work..

`./gateway-api-controller -metrics-bind-address :8083 -kubeconfig ~/.kube/config`

Want to change the gatewayClass then the flag `-gateway-class-name` will probably help

### Example

The `/manifests` folder contains the basics of the `GatewayClass` and `Gateway` yaml structure..

## Implented logic

Currently the `GatewayClass` will set the status `ACCEPTED -> True` if the gatway controller matches and `Gateway` will verify that the parent `GatewayClass` exists.. that's it so far (clearly a long way to go)

## Want to Contribute?

Please and thankyou