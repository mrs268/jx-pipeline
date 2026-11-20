package pipelines

import (
	"fmt"
	"testing"

	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/stretchr/testify/require"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestAddTaskRunsMessage(t *testing.T) {
	tests := []struct {
		name     string
		kinds    []v1.ActivityStepKindType
		taskRuns int
	}{
		{name: "no task runs", kinds: []v1.ActivityStepKindType{v1.ActivityStepKindTypeStage}},
		{name: "no steps", taskRuns: 1},
		{name: "stage", kinds: []v1.ActivityStepKindType{v1.ActivityStepKindTypeStage}, taskRuns: 1},
		{name: "preview", kinds: []v1.ActivityStepKindType{v1.ActivityStepKindTypePreview}, taskRuns: 1},
		{name: "promote", kinds: []v1.ActivityStepKindType{v1.ActivityStepKindTypePromote}, taskRuns: 1},
		{
			name: "mixed steps and excess task runs",
			kinds: []v1.ActivityStepKindType{
				v1.ActivityStepKindTypeStage,
				v1.ActivityStepKindTypePreview,
				v1.ActivityStepKindTypePromote,
				v1.ActivityStepKindTypeStage,
			},
			taskRuns: 5,
		},
		{
			name:     "fewer task runs than stages",
			kinds:    []v1.ActivityStepKindType{v1.ActivityStepKindTypeStage, v1.ActivityStepKindTypeStage},
			taskRuns: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pa := &v1.PipelineActivity{}
			for _, kind := range tt.kinds {
				step := v1.PipelineActivityStep{Kind: kind}
				switch kind {
				case v1.ActivityStepKindTypeStage:
					step.Stage = &v1.StageActivityStep{CoreActivityStep: v1.CoreActivityStep{Message: "original"}}
				case v1.ActivityStepKindTypePreview:
					step.Preview = &v1.PreviewActivityStep{}
				case v1.ActivityStepKindTypePromote:
					step.Promote = &v1.PromoteActivityStep{}
				}
				pa.Spec.Steps = append(pa.Spec.Steps, step)
			}

			pr := &v1beta1.PipelineRun{}
			pr.Status.TaskRuns = make(map[string]*v1beta1.PipelineRunTaskRunStatus)
			for i := 0; i < tt.taskRuns; i++ {
				pr.Status.TaskRuns[fmt.Sprintf("task-%d", i)] = &v1beta1.PipelineRunTaskRunStatus{
					Status: &v1beta1.TaskRunStatus{
						Status: duckv1beta1.Status{
							Conditions: duckv1beta1.Conditions{{Message: "TaskRun completed"}},
						},
					},
				}
			}

			require.NotPanics(t, func() { addTaskRunsMessage(pr, pa) })

			for i, step := range pa.Spec.Steps {
				if step.Stage == nil {
					continue
				}
				if i < tt.taskRuns {
					require.Equal(t, v1.ActivityMessageType("Stage completed"), step.Stage.Message)
				} else {
					require.Equal(t, v1.ActivityMessageType("original"), step.Stage.Message)
				}
			}
		})
	}
}
