package ai

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// chatBedrock calls the Bedrock Converse API (Claude, Nova, Llama, etc. on Bedrock).
func chatBedrock(ctx context.Context, cfg ProviderSettings, system string, messages []Message) (string, error) {
	region := strings.TrimSpace(cfg.APIURL)
	if region == "" {
		return "", fmt.Errorf("AWS Bedrock: region not configured")
	}

	awscfg, err := loadBedrockAWSConfig(ctx, region)
	if err != nil {
		return "", err
	}
	client := bedrockruntime.NewFromConfig(awscfg)

	var brMessages []types.Message
	for _, m := range messages {
		role := types.ConversationRoleUser
		if m.Role == "assistant" {
			role = types.ConversationRoleAssistant
		}
		brMessages = append(brMessages, types.Message{
			Role: role,
			Content: []types.ContentBlock{
				&types.ContentBlockMemberText{Value: m.Content},
			},
		})
	}

	modelID := bedrockConverseModelID(cfg.Model)
	input := &bedrockruntime.ConverseInput{
		ModelId:  aws.String(modelID),
		Messages: brMessages,
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens: aws.Int32(bedrockMaxTokensInt32(cfg.MaxTokens)),
		},
	}
	if system != "" {
		input.System = []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{Value: system},
		}
	}

	out, err := client.Converse(ctx, input)
	if err != nil {
		return "", fmt.Errorf("AWS Bedrock: %w", err)
	}
	if out.Output == nil {
		return "", fmt.Errorf("AWS Bedrock: empty response")
	}
	msg, ok := out.Output.(*types.ConverseOutputMemberMessage)
	if !ok {
		return "", fmt.Errorf("AWS Bedrock: unexpected output type")
	}
	var parts []string
	appendBedrockOutputParts(&parts, msg.Value.Content)
	if len(parts) == 0 {
		return "", fmt.Errorf("AWS Bedrock: empty message content")
	}
	return strings.TrimSpace(strings.Join(parts, "\n")), nil
}

// appendBedrockOutputParts collects assistant-visible text from Converse output blocks.
// Reasoning models (e.g. Claude 3.7+ with extended thinking) may return reasoningContent
// alongside or before text; we surface reasoning text when plain text is absent or sparse.
func appendBedrockOutputParts(parts *[]string, blocks []types.ContentBlock) {
	for _, block := range blocks {
		switch b := block.(type) {
		case *types.ContentBlockMemberText:
			if s := strings.TrimSpace(b.Value); s != "" {
				*parts = append(*parts, s)
			}
		case *types.ContentBlockMemberReasoningContent:
			appendReasoningContentParts(parts, b.Value)
		default:
			// toolUse, toolResult, image, etc. — not mapped to chat text here
		}
	}
}

func appendReasoningContentParts(parts *[]string, rc types.ReasoningContentBlock) {
	if rc == nil {
		return
	}
	switch r := rc.(type) {
	case *types.ReasoningContentBlockMemberReasoningText:
		if r.Value.Text != nil {
			if s := strings.TrimSpace(*r.Value.Text); s != "" {
				*parts = append(*parts, s)
			}
		}
	case *types.ReasoningContentBlockMemberRedactedContent:
		// No human-readable text; skip
	default:
	}
}

// bedrockMaxTokensInt32 maps configured max tokens to int32 without overflow (G115).
func bedrockMaxTokensInt32(maxTokens int) int32 {
	if maxTokens <= 0 {
		return 1
	}
	if maxTokens > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(maxTokens)
}

func loadBedrockAWSConfig(ctx context.Context, region string) (aws.Config, error) {
	// Amazon Bedrock API keys (console “Generate API key”) use bearer auth, not SigV4.
	// The AWS SDK reads AWS_BEARER_TOKEN_BEDROCK automatically when set.
	if key := bedrockAPIKeyFromEnv(); key != "" {
		_ = os.Setenv("AWS_BEARER_TOKEN_BEDROCK", key)
		return awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	}

	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(region))

	accessKey, secretKey, sessionToken := bedrockIAMCredentialEnv()
	if accessKey != "" && secretKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken),
		))
	}

	return awsconfig.LoadDefaultConfig(ctx, opts...)
}

// bedrockAPIKeyFromEnv returns a Bedrock bearer API key (not IAM access key id).
func bedrockAPIKeyFromEnv() string {
	if k := strings.TrimSpace(os.Getenv("REAPER_AI_BEDROCK_API_KEY")); k != "" {
		return k
	}
	return strings.TrimSpace(os.Getenv("AWS_BEARER_TOKEN_BEDROCK"))
}

func bedrockIAMCredentialEnv() (accessKey, secretKey, sessionToken string) {
	accessKey = strings.TrimSpace(os.Getenv("REAPER_AI_BEDROCK_ACCESS_KEY_ID"))
	secretKey = strings.TrimSpace(os.Getenv("REAPER_AI_BEDROCK_SECRET_ACCESS_KEY"))
	sessionToken = strings.TrimSpace(os.Getenv("REAPER_AI_BEDROCK_SESSION_TOKEN"))
	if accessKey == "" {
		accessKey = strings.TrimSpace(os.Getenv("AWS_ACCESS_KEY_ID"))
	}
	if secretKey == "" {
		secretKey = strings.TrimSpace(os.Getenv("AWS_SECRET_ACCESS_KEY"))
	}
	if sessionToken == "" {
		sessionToken = strings.TrimSpace(os.Getenv("AWS_SESSION_TOKEN"))
	}
	return accessKey, secretKey, sessionToken
}

func bedrockRegionFromEnv() string {
	if r := strings.TrimSpace(os.Getenv("REAPER_AI_BEDROCK_REGION")); r != "" {
		return r
	}
	if r := strings.TrimSpace(os.Getenv("AWS_REGION")); r != "" {
		return r
	}
	return strings.TrimSpace(os.Getenv("AWS_DEFAULT_REGION"))
}

func bedrockHasAPIKey() bool {
	return bedrockAPIKeyFromEnv() != ""
}

func bedrockHasIAMAccessKeys() bool {
	ak, sk, _ := bedrockIAMCredentialEnv()
	return ak != "" && sk != ""
}

func bedrockCanAuthenticate() bool {
	return bedrockHasAPIKey() || bedrockHasIAMAccessKeys() || bedrockUseIAM()
}

func bedrockUseIAM() bool {
	if bedrockHasAPIKey() {
		return false
	}
	if strings.TrimSpace(os.Getenv("REAPER_AI_BEDROCK_USE_IAM")) == "1" {
		return true
	}
	// Enabled without static keys: default credential chain (EKS IRSA, instance profile, etc.).
	if strings.TrimSpace(os.Getenv("REAPER_AI_BEDROCK_ENABLED")) == "1" && !bedrockHasIAMAccessKeys() {
		return true
	}
	return false
}
