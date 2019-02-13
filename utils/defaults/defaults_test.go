package defaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
)

func TestGetRolloutReplicasOrDefault(t *testing.T) {
	replicas := int32(2)
	rolloutNonDefaultValue := &v1alpha1.Rollout{
		Spec: v1alpha1.RolloutSpec{
			Replicas: &replicas,
		},
	}

	assert.Equal(t, replicas, GetRolloutReplicasOrDefault(rolloutNonDefaultValue))
	rolloutDefaultValue := &v1alpha1.Rollout{}
	assert.Equal(t, DefaultReplicas, GetRolloutReplicasOrDefault(rolloutDefaultValue))
}

func TestGetRevisionHistoryOrDefault(t *testing.T) {
	revisionHistoryLimit := int32(2)
	rolloutNonDefaultValue := &v1alpha1.Rollout{
		Spec: v1alpha1.RolloutSpec{
			RevisionHistoryLimit: &revisionHistoryLimit,
		},
	}

	assert.Equal(t, revisionHistoryLimit, GetRevisionHistoryLimitOrDefault(rolloutNonDefaultValue))
	rolloutDefaultValue := &v1alpha1.Rollout{}
	assert.Equal(t, DefaultRevisionHistoryLimit, GetRevisionHistoryLimitOrDefault(rolloutDefaultValue))
}

func TestGetMaxSurgeOrDefault(t *testing.T) {
	maxSurge := intstr.FromInt(2)
	rolloutNonDefaultValue := &v1alpha1.Rollout{
		Spec: v1alpha1.RolloutSpec{
			Strategy: v1alpha1.RolloutStrategy{
				CanaryStrategy: &v1alpha1.CanaryStrategy{
					MaxSurge: &maxSurge,
				},
			},
		},
	}

	assert.Equal(t, maxSurge, *GetMaxSurgeOrDefault(rolloutNonDefaultValue))
	rolloutDefaultValue := &v1alpha1.Rollout{}
	assert.Equal(t, intstr.FromInt(DefaultMaxSurge), *GetMaxSurgeOrDefault(rolloutDefaultValue))
}

func TestGetMaxUnavailableOrDefault(t *testing.T) {
	maxUnavailable := intstr.FromInt(2)
	rolloutNonDefaultValue := &v1alpha1.Rollout{
		Spec: v1alpha1.RolloutSpec{
			Strategy: v1alpha1.RolloutStrategy{
				CanaryStrategy: &v1alpha1.CanaryStrategy{
					MaxUnavailable: &maxUnavailable,
				},
			},
		},
	}

	assert.Equal(t, maxUnavailable, *GetMaxUnavailableOrDefault(rolloutNonDefaultValue))
	rolloutDefaultValue := &v1alpha1.Rollout{}
	assert.Equal(t, intstr.FromInt(DefaultMaxUnavailable), *GetMaxUnavailableOrDefault(rolloutDefaultValue))
}
