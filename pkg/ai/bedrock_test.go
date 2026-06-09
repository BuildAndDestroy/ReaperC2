package ai

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

func TestBedrockMaxTokensInt32(t *testing.T) {
	if got := bedrockMaxTokensInt32(2048); got != 2048 {
		t.Fatalf("got %d", got)
	}
	if got := bedrockMaxTokensInt32(0); got != 1 {
		t.Fatalf("zero got %d", got)
	}
	if got := bedrockMaxTokensInt32(1<<62); got != 2147483647 {
		t.Fatalf("overflow got %d", got)
	}
}

func TestAppendBedrockOutputParts_reasoningOnly(t *testing.T) {
	var parts []string
	blocks := []types.ContentBlock{
		&types.ContentBlockMemberReasoningContent{
			Value: &types.ReasoningContentBlockMemberReasoningText{
				Value: types.ReasoningTextBlock{Text: aws.String("  chain-of-thought  ")},
			},
		},
	}
	appendBedrockOutputParts(&parts, blocks)
	if len(parts) != 1 || parts[0] != "chain-of-thought" {
		t.Fatalf("got %#v", parts)
	}
}

func TestAppendBedrockOutputParts_textAndReasoning(t *testing.T) {
	var parts []string
	blocks := []types.ContentBlock{
		&types.ContentBlockMemberReasoningContent{
			Value: &types.ReasoningContentBlockMemberReasoningText{
				Value: types.ReasoningTextBlock{Text: aws.String("think")},
			},
		},
		&types.ContentBlockMemberText{Value: "answer"},
	}
	appendBedrockOutputParts(&parts, blocks)
	if len(parts) != 2 || parts[0] != "think" || parts[1] != "answer" {
		t.Fatalf("got %#v", parts)
	}
}
