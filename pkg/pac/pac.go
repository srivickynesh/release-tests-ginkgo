package pac

import (
	"fmt"
	"log"
	"os"

	"github.com/srivickynesh/release-tests-ginkgo/pkg/oc"
	"github.com/srivickynesh/release-tests-ginkgo/pkg/store"
	"github.com/xanzy/go-gitlab"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	webhookConfigName = "gitlab-webhook-config"
)

var client *gitlab.Client

func SetGitLabClient(c *gitlab.Client) {
	client = c
}

// Initialize Gitlab Client
func InitGitLabClient() *gitlab.Client {
	tokenSecretData := os.Getenv("GITLAB_TOKEN")
	webhookSecretData := os.Getenv("GITLAB_WEBHOOK_TOKEN")
	if tokenSecretData == "" && webhookSecretData == "" {
		Fail(fmt.Sprintf("token for authorization to the GitLab repository was not exported as a system variable"))
	} else {
		if !oc.SecretExists(webhookConfigName, store.Namespace()) {
			oc.CreateSecretForWebhook(tokenSecretData, webhookSecretData, store.Namespace())
		} else {
			log.Printf("Secret \"%s\" already exists", webhookConfigName)
		}
	}
	client, err := gitlab.NewClient(tokenSecretData)
	if err != nil {
		Fail(fmt.Sprintf("failed to initialize GitLab client: %v", err))
	}

	return client
}
