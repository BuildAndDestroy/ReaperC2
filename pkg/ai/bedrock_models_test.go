package ai

import "testing"

func TestBedrockConverseModelIDInferenceProfile(t *testing.T) {
	t.Setenv("REAPER_AI_BEDROCK_REGION", "us-east-1")
	t.Setenv("REAPER_AI_BEDROCK_INFERENCE_PREFIX", "")

	got := bedrockConverseModelID("anthropic.claude-opus-4-7")
	if got != "us.anthropic.claude-opus-4-7" {
		t.Fatalf("got %q", got)
	}
	got = bedrockConverseModelID("anthropic.claude-sonnet-4-6")
	if got != "us.anthropic.claude-sonnet-4-6" {
		t.Fatalf("got %q", got)
	}
}

func TestBedrockConverseModelIDPassthrough(t *testing.T) {
	if got := bedrockConverseModelID("us.anthropic.claude-opus-4-7"); got != "us.anthropic.claude-opus-4-7" {
		t.Fatalf("got %q", got)
	}
	if got := bedrockConverseModelID("amazon.nova-lite-v1:0"); got != "amazon.nova-lite-v1:0" {
		t.Fatalf("got %q", got)
	}
}

func TestBedrockConverseModelIDClaudeFable5GlobalProfile(t *testing.T) {
	t.Setenv("REAPER_AI_BEDROCK_REGION", "us-east-1")
	if got := bedrockConverseModelID("anthropic.claude-fable-5"); got != "global.anthropic.claude-fable-5" {
		t.Fatalf("got %q", got)
	}
	if got := bedrockConverseModelID("global.anthropic.claude-fable-5"); got != "global.anthropic.claude-fable-5" {
		t.Fatalf("got %q", got)
	}
}

func TestBedrockInferenceProfilePrefixEU(t *testing.T) {
	t.Setenv("REAPER_AI_BEDROCK_REGION", "eu-west-1")
	t.Setenv("REAPER_AI_BEDROCK_INFERENCE_PREFIX", "")

	got := bedrockConverseModelID("anthropic.claude-sonnet-4-6")
	if got != "eu.anthropic.claude-sonnet-4-6" {
		t.Fatalf("got %q", got)
	}
}

func TestBedrockModelNamesMergedResolvesProfiles(t *testing.T) {
	t.Setenv("REAPER_AI_BEDROCK_REGION", "us-east-1")
	t.Setenv("REAPER_AI_BEDROCK_MODELS", "anthropic.claude-opus-4-7,amazon.nova-lite-v1:0")
	t.Setenv("REAPER_AI_BEDROCK_MODEL", "")

	names := bedrockModelNamesMerged()
	if len(names) != 2 {
		t.Fatalf("names = %#v", names)
	}
	if names[0] != "us.anthropic.claude-opus-4-7" {
		t.Fatalf("first = %q", names[0])
	}
}
