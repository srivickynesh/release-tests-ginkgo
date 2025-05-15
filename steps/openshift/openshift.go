package openshift

import (
	"github.com/getgauge-contrib/gauge-go/gauge"
	"github.com/srivickynesh/release-tests-ginkgo/pkg/openshift"
	"github.com/srivickynesh/release-tests-ginkgo/pkg/store"
	"github.com/srivickynesh/release-tests-ginkgo/pkg/triggers"
)

var _ = gauge.Step("Get tags of the imagestream <imageStream> from namespace <namespace> and store to variable <variableName>", func(imageStream, namespace, variableName string) {
	tagNames := openshift.GetImageStreamTags(store.Clients(), namespace, imageStream)
	store.PutScenarioDataSlice(variableName, tagNames)
})

var _ = gauge.Step("Verify that image stream <is> exists", func(is string) {
	openshift.VerifyImageStreamExists(store.Clients(), is, "openshift")
})

var _ = gauge.Step("Get route url of the route <routeName>", func(routeName string) {
	routeurl := triggers.GetRouteURL(routeName, store.Namespace())
	store.PutScenarioData("routeurl", routeurl)
})
