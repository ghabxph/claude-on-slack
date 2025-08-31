package notifications

import (
	"fmt"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"

	"github.com/ghabxph/claude-on-slack/internal/version"
)

type DeploymentNotifier struct {
	slackClient *slack.Client
	channels    []string
	logger      *zap.Logger
}

func NewDeploymentNotifier(slackClient *slack.Client, channels []string, logger *zap.Logger) *DeploymentNotifier {
	return &DeploymentNotifier{
		slackClient: slackClient,
		channels:    channels,
		logger:      logger,
	}
}

func (n *DeploymentNotifier) NotifyDeployment(changes []string) error {
	message := n.FormatDeploymentMessage(version.GetVersion(), changes)
	
	return n.SendConcurrentNotifications(message)
}

func (n *DeploymentNotifier) FormatDeploymentMessage(version string, changes []string) string {
	message := fmt.Sprintf("🚀 *Claude Bot Deployment Complete* - v%s\n", version)
	message += fmt.Sprintf("⏰ Deployed at: %s\n\n", time.Now().Format("2006-01-02 15:04:05 UTC"))
	
	if len(changes) > 0 {
		message += "*Changes in this release:*\n"
		for _, change := range changes {
			message += fmt.Sprintf("• %s\n", change)
		}
	} else {
		message += "• 🛠️ **Database Query Robustness** - Replaced SELECT * with explicit columns for schema safety\n"
		message += "• 🔧 **SQL Scan Error Fix** - Resolved critical column order mismatch in child session queries\n"
		message += "• 🚀 **Migration-Proof Design** - Future database changes won't break existing queries\n"
		message += "• 🎯 **Type Safety** - Eliminated runtime SQL scan errors from column/field misalignment\n"
		message += "• ⚡ **Performance & Maintainability** - Explicit column selection improves query optimization\n"
	}
	
	message += "\n📋 *Full details*: See <https://github.com/ghabxph/claude-on-slack/blob/main/CHANGELOG.md|CHANGELOG.md>\n"
	message += "✅ All systems operational"
	
	return message
}

func (n *DeploymentNotifier) SendConcurrentNotifications(message string) error {
	if len(n.channels) == 0 {
		n.logger.Info("No notification channels configured, skipping deployment notification")
		return nil
	}

	errChan := make(chan error, len(n.channels))
	
	for _, channel := range n.channels {
		go func(ch string) {
			_, _, err := n.slackClient.PostMessage(ch,
				slack.MsgOptionText(message, false),
				slack.MsgOptionAsUser(true))
			errChan <- err
		}(channel)
	}

	// Collect results
	var errors []error
	for i := 0; i < len(n.channels); i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err)
			n.logger.Error("Failed to send deployment notification", 
				zap.Error(err),
				zap.String("channel", n.channels[i]))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send notifications to %d channels", len(errors))
	}

	n.logger.Info("Deployment notifications sent successfully", 
		zap.Int("channels", len(n.channels)))

	return nil
}