package main

import (
	"knative.dev/eventing-contrib/azsb/channel/pkg/reconciler/controller"
	"knative.dev/pkg/injection/sharedmain"
)

const component = "azsbchannel_controller"

func main() {
	sharedmain.Main(component, controller.NewController)
}
