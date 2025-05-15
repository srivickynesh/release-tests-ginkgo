package pac

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/srivickynesh/release-tests-ginkgo/pkg/k8s"
	"github.com/srivickynesh/release-tests-ginkgo/pkg/pac"
	"github.com/srivickynesh/release-tests-ginkgo/pkg/pipelines"
	"github.com/srivickynesh/release-tests-ginkgo/pkg/store"
)

var _ = Describe("Pipelines As Code tests", func() {

	Describe("Configure PAC in GitLab Project: PIPELINES-30-TC01", func() {
		It("Setup Gitlab Client", func() {
			c := pac.InitGitLabClient()
			pac.SetGitLabClient(c)
		})

		It("Validate PAC Info Install", func() {
			pac.AssertPACInfoInstall()
		})

		It("Create Smee deployment", func() {
			pac.SetupSmeeDeployment()
			k8s.ValidateDeployments(store.Clients(), store.Namespace(), store.GetScenarioData("smeeDeploymentName"))
		})

		It("Configure GitLab repo for \"pull_request\" in \"main\"", func() {
			pac.SetupGitLabProject()
			pac.GeneratePipelineRunYaml("pull_request", "main")
		})

		It("Configure PipelineRun", func() {
			pac.ConfigurePreviewChanges()
		})

		It("Validate PipelineRun for \"success\"", func() {
			pipelineName := pac.GetPipelineNameFromMR()
			pipelines.ValidatePipelineRun(store.Clients(), pipelineName, "success", "no", store.Namespace())
		})

		It("Cleanup PAC", func() {
			pac.CleanupPAC(store.Clients(), store.GetScenarioData("smeeDeploymentName"), store.Namespace())
		})
	})

	Describe("Configure PAC in GitLab Project: PIPELINES-30-TC02", func() {
		It("Setup Gitlab Client", func() {
			c := pac.InitGitLabClient()
			pac.SetGitLabClient(c)
		})

		It("Create Smee deployment", func() {
			pac.SetupSmeeDeployment()
			k8s.ValidateDeployments(store.Clients(), store.Namespace(), store.GetScenarioData("smeeDeploymentName"))
		})

		It("Configure GitLab repo for \"pull_request\" in \"main\"", func() {
			pac.SetupGitLabProject()
			pac.GeneratePipelineRunYaml("pull_request", "main")
		})

		It("Update Annotation", func() {
			pac.UpdateAnnotation("pipelinesascode.tekton.dev/on-label", "[bug]")
		})

		It("Configure PipelineRun", func() {
			pac.ConfigurePreviewChanges()
		})

		It("Check 0 pipelineruns present within 10 seconds", func() {
			// Implement a wait/poll function or validate manually
		})

		It("Add Label", func() {
			pac.AddLabel("bug", "red", "Identify a Issue")
		})

		It("Validate PipelineRun for \"success\"", func() {
			pipelineName := pac.GetPipelineNameFromMR()
			pipelines.ValidatePipelineRun(store.Clients(), pipelineName, "success", "no", store.Namespace())
		})

		It("Cleanup PAC", func() {
			pac.CleanupPAC(store.Clients(), store.GetScenarioData("smeeDeploymentName"), store.Namespace())
		})
	})

	Describe("Configure PAC in GitLab Project: PIPELINES-30-TC03", func() {
		It("Setup Gitlab Client", func() {
			c := pac.InitGitLabClient()
			pac.SetGitLabClient(c)
		})

		It("Create Smee deployment", func() {
			pac.SetupSmeeDeployment()
			k8s.ValidateDeployments(store.Clients(), store.Namespace(), store.GetScenarioData("smeeDeploymentName"))
		})

		It("Configure GitLab repo for \"pull_request\" in \"main\"", func() {
			pac.SetupGitLabProject()
			pac.GeneratePipelineRunYaml("pull_request", "main")
		})

		It("Update Annotation", func() {
			pac.UpdateAnnotation("pipelinesascode.tekton.dev/on-comment", "^/hello-world")
		})

		It("Configure PipelineRun", func() {
			pac.ConfigurePreviewChanges()
		})

		It("Validate PipelineRun for \"success\"", func() {
			pipelineName := pac.GetPipelineNameFromMR()
			pipelines.ValidatePipelineRun(store.Clients(), pipelineName, "success", "no", store.Namespace())
		})

		It("Add Comment", func() {
			pac.AddComment("/hello-world")
		})

		It("Check 2 pipelineruns present within 10 seconds", func() {
			// Implement a wait/poll function or validate manually
		})

		It("Validate PipelineRun for \"success\"", func() {
			pipelineName := pac.GetPipelineNameFromMR()
			pipelines.ValidatePipelineRun(store.Clients(), pipelineName, "success", "no", store.Namespace())
		})

		It("Cleanup PAC", func() {
			pac.CleanupPAC(store.Clients(), store.GetScenarioData("smeeDeploymentName"), store.Namespace())
		})
	})
})
