package service

import (
	"testing"

	assistantdto "github.com/han139242/course-recommend-agents/backend/internal/module/assistant/dto"
)

func TestInferClientActions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		message     string
		wantType    string
		wantTarget  string
		wantCourse  string
		wantActions int
	}{
		{
			name:        "focus knowledge",
			message:     "\u805a\u7126 \u5f3a\u5316\u5b66\u4e60",
			wantType:    "focus_knowledge",
			wantTarget:  "\u5f3a\u5316\u5b66\u4e60",
			wantActions: 1,
		},
		{
			name:        "expand knowledge",
			message:     "\u5c55\u5f00 \u5f3a\u5316\u5b66\u4e60 \u7684\u5173\u8054\u8282\u70b9",
			wantType:    "expand_knowledge",
			wantTarget:  "\u5f3a\u5316\u5b66\u4e60",
			wantActions: 1,
		},
		{
			name:        "highlight prerequisite chain",
			message:     "\u9ad8\u4eae \u6a21\u578b\u8bc4\u4f30 \u7684\u524d\u7f6e\u94fe",
			wantType:    "highlight_prerequisite_chain",
			wantTarget:  "\u6a21\u578b\u8bc4\u4f30",
			wantActions: 1,
		},
		{
			name:        "view prerequisite chain naturally",
			message:     "\u770b\u4e00\u4e0b\u5f3a\u5316\u5b66\u4e60\u7684\u524d\u7f6e\u77e5\u8bc6\u94fe",
			wantType:    "highlight_prerequisite_chain",
			wantTarget:  "\u5f3a\u5316\u5b66\u4e60",
			wantActions: 1,
		},
		{
			name:        "open knowledge detail",
			message:     "\u6253\u5f00 \u6df1\u5ea6\u5b66\u4e60 \u7684\u8be6\u60c5\u9875",
			wantType:    "open_knowledge_detail",
			wantTarget:  "\u6df1\u5ea6\u5b66\u4e60",
			wantActions: 1,
		},
		{
			name:        "filter course",
			message:     "\u6309\u8bfe\u7a0b\u7b5b\u9009 \u77e5\u8bc6\u56fe\u8c31",
			wantType:    "filter_course",
			wantCourse:  "\u77e5\u8bc6\u56fe\u8c31",
			wantActions: 1,
		},
		{
			name:        "filter course legacy alias",
			message:     "\u6309\u8bfe\u7a0b\u7b5b\u9009 \u7eea\u8bba",
			wantType:    "filter_course",
			wantCourse:  "\u4eba\u5de5\u667a\u80fd\u5bfc\u8bba",
			wantActions: 1,
		},
		{
			name:        "switch graph to course naturally",
			message:     "\u5e2e\u6211\u628a\u56fe\u8c31\u5207\u5230\u5f3a\u5316\u5b66\u4e60",
			wantType:    "filter_course",
			wantCourse:  "\u5f3a\u5316\u5b66\u4e60",
			wantActions: 1,
		},
		{
			name:        "fit view",
			message:     "\u9002\u914d\u89c6\u56fe",
			wantType:    "fit_graph_view",
			wantActions: 1,
		},
		{
			name:        "reset scene",
			message:     "\u91cd\u7f6e\u573a\u666f",
			wantType:    "reset_graph_scene",
			wantActions: 1,
		},
		{
			name:        "compound expand and focus",
			message:     "\u5c55\u5f00 Transformer \u7684\u5173\u8054\u8282\u70b9\u5e76\u805a\u7126\u5230\u56fe\u91cc",
			wantType:    "expand_knowledge",
			wantTarget:  "Transformer",
			wantActions: 2,
		},
		{
			name:        "zoom graph and focus",
			message:     "\u8bf7\u5e2e\u6211\u7f29\u653e\u56fe\u8c31\u5e76\u805a\u7126\u673a\u5668\u5b66\u4e60",
			wantType:    "fit_graph_view",
			wantActions: 2,
		},
		{
			name:        "non control question",
			message:     "\u5e2e\u6211\u89e3\u91ca \u5f3a\u5316\u5b66\u4e60",
			wantActions: 0,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			actions := inferClientActions([]map[string]any{
				{"role": "user", "content": testCase.message},
			})

			if len(actions) != testCase.wantActions {
				t.Fatalf("expected %d actions, got %d", testCase.wantActions, len(actions))
			}
			if testCase.wantActions == 0 {
				return
			}

			if actions[0].Type != testCase.wantType {
				t.Fatalf("expected action type %q, got %q", testCase.wantType, actions[0].Type)
			}
			if testCase.wantTarget != "" && actions[0].Target != testCase.wantTarget {
				t.Fatalf("expected target %q, got %q", testCase.wantTarget, actions[0].Target)
			}
			if testCase.wantCourse != "" && actions[0].CourseTitle != testCase.wantCourse {
				t.Fatalf("expected course %q, got %q", testCase.wantCourse, actions[0].CourseTitle)
			}
			if testCase.name == "compound expand and focus" {
				if actions[1].Type != "focus_knowledge" {
					t.Fatalf("expected second action type %q, got %q", "focus_knowledge", actions[1].Type)
				}
				if actions[1].Target != "Transformer" {
					t.Fatalf("expected second action target %q, got %q", "Transformer", actions[1].Target)
				}
			}
			if testCase.name == "zoom graph and focus" {
				if actions[1].Type != "focus_knowledge" {
					t.Fatalf("expected second action type %q, got %q", "focus_knowledge", actions[1].Type)
				}
				if actions[1].Target != "机器学习" {
					t.Fatalf("expected second action target %q, got %q", "机器学习", actions[1].Target)
				}
			}
		})
	}
}

func TestBuildActionFallbackReplySuppressesGenericModelContent(t *testing.T) {
	t.Parallel()

	reply := buildActionFallbackReply([]assistantdto.ClientAction{
		{
			Type:   "focus_knowledge",
			Target: "\u5f3a\u5316\u5b66\u4e60",
			Flash:  true,
		},
	}, "\u6211\u53ef\u4ee5\u5e2e\u60a8\u63a2\u7d22\u77e5\u8bc6\u56fe\u8c31\uff0c\u8bf7\u544a\u8bc9\u6211\u60a8\u611f\u5174\u8da3\u7684\u4e3b\u9898\u3002")

	if reply != "\u5df2\u4e3a\u4f60\u5c1d\u8bd5\u805a\u7126\u5230\u201c\u5f3a\u5316\u5b66\u4e60\u201d\u3002" {
		t.Fatalf("unexpected fallback reply: %q", reply)
	}
}

func TestInferClientActionsProposesGoalProfile(t *testing.T) {
	t.Parallel()

	actions := inferClientActions([]map[string]any{
		{"role": "user", "content": "\u6211\u662f\u5927\u4e8c\u5b66\u751f\uff0c\u60f3\u505a\u667a\u80fd\u4f53\u9879\u76ee\uff0c\u5b66\u8fc7 Python \u548c\u4e00\u70b9\u673a\u5668\u5b66\u4e60\uff0c\u6570\u5b66\u57fa\u7840\u4e00\u822c"},
	})

	if len(actions) != 1 {
		t.Fatalf("expected one action, got %d", len(actions))
	}
	if actions[0].Type != "propose_goal_profile" {
		t.Fatalf("expected propose_goal_profile, got %q", actions[0].Type)
	}
	if actions[0].Payload["gradeLevel"] != "sophomore" {
		t.Fatalf("expected sophomore profile, got %#v", actions[0].Payload)
	}
	if actions[0].Payload["goalType"] != "project_building" {
		t.Fatalf("expected project goal, got %#v", actions[0].Payload)
	}
}

func TestShouldHandleActionsDirectly(t *testing.T) {
	t.Parallel()

	directActions := []assistantdto.ClientAction{
		{
			Type:   "focus_knowledge",
			Target: "\u5f3a\u5316\u5b66\u4e60",
			Flash:  true,
		},
	}

	if !shouldHandleActionsDirectly([]map[string]any{
		{"role": "user", "content": "\u805a\u7126 \u5f3a\u5316\u5b66\u4e60"},
	}, directActions) {
		t.Fatal("expected direct action command to bypass model")
	}

	if shouldHandleActionsDirectly([]map[string]any{
		{"role": "user", "content": "\u5e2e\u6211\u89e3\u91ca \u5f3a\u5316\u5b66\u4e60 \u5e76\u805a\u7126\u5230\u5b83"},
	}, directActions) {
		t.Fatal("expected mixed explanation request to keep model in the loop")
	}
}
