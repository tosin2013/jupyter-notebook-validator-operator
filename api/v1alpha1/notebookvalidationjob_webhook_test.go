/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"testing"
)

// TestValidateVolumes_EmptyVolumes tests that empty volumes pass validation
func TestValidateVolumes_EmptyVolumes(t *testing.T) {
	job := &NotebookValidationJob{
		Spec: NotebookValidationJobSpec{
			PodConfig: PodConfigSpec{
				Volumes:      nil,
				VolumeMounts: nil,
			},
		},
	}

	err := validateVolumes(job)
	if err != nil {
		t.Errorf("Expected no error for empty volumes, got: %v", err)
	}
}

// TestValidateVolumes_ValidPVCVolume tests a valid PVC volume configuration
func TestValidateVolumes_ValidPVCVolume(t *testing.T) {
	job := &NotebookValidationJob{
		Spec: NotebookValidationJobSpec{
			PodConfig: PodConfigSpec{
				Volumes: []Volume{
					{
						Name: "model-output",
						PersistentVolumeClaim: &PersistentVolumeClaimVolumeSource{
							ClaimName: "trained-models-pvc",
						},
					},
				},
				VolumeMounts: []VolumeMount{
					{
						Name:      "model-output",
						MountPath: "/models",
					},
				},
			},
		},
	}

	err := validateVolumes(job)
	if err != nil {
		t.Errorf("Expected no error for valid PVC volume, got: %v", err)
	}
}

// TestValidateVolumes_ValidConfigMapVolume tests a valid ConfigMap volume configuration
func TestValidateVolumes_ValidConfigMapVolume(t *testing.T) {
	job := &NotebookValidationJob{
		Spec: NotebookValidationJobSpec{
			PodConfig: PodConfigSpec{
				Volumes: []Volume{
					{
						Name: "config",
						ConfigMap: &ConfigMapVolumeSource{
							Name: "notebook-config",
						},
					},
				},
				VolumeMounts: []VolumeMount{
					{
						Name:      "config",
						MountPath: "/config",
					},
				},
			},
		},
	}

	err := validateVolumes(job)
	if err != nil {
		t.Errorf("Expected no error for valid ConfigMap volume, got: %v", err)
	}
}

// TestValidateVolumes_ValidSecretVolume tests a valid Secret volume configuration
func TestValidateVolumes_ValidSecretVolume(t *testing.T) {
	job := &NotebookValidationJob{
		Spec: NotebookValidationJobSpec{
			PodConfig: PodConfigSpec{
				Volumes: []Volume{
					{
						Name: "certs",
						Secret: &SecretVolumeSource{
							SecretName: "model-endpoint-certs",
						},
					},
				},
				VolumeMounts: []VolumeMount{
					{
						Name:      "certs",
						MountPath: "/certs",
						ReadOnly:  true,
					},
				},
			},
		},
	}

	err := validateVolumes(job)
	if err != nil {
		t.Errorf("Expected no error for valid Secret volume, got: %v", err)
	}
}

// TestValidateVolumes_ValidEmptyDirVolume tests a valid EmptyDir volume configuration
func TestValidateVolumes_ValidEmptyDirVolume(t *testing.T) {
	job := &NotebookValidationJob{
		Spec: NotebookValidationJobSpec{
			PodConfig: PodConfigSpec{
				Volumes: []Volume{
					{
						Name: "scratch",
						EmptyDir: &EmptyDirVolumeSource{
							SizeLimit: "10Gi",
						},
					},
				},
				VolumeMounts: []VolumeMount{
					{
						Name:      "scratch",
						MountPath: "/tmp/scratch",
					},
				},
			},
		},
	}

	err := validateVolumes(job)
	if err != nil {
		t.Errorf("Expected no error for valid EmptyDir volume, got: %v", err)
	}
}

// TestValidateVolumes_ReservedVolumeName tests that reserved volume names are rejected
func TestValidateVolumes_ReservedVolumeName(t *testing.T) {
	testCases := []struct {
		name         string
		volumeName   string
		expectError  bool
		errorContain string
	}{
		{"git-clone reserved", "git-clone", true, "reserved for internal use"},
		{"notebook-data reserved", "notebook-data", true, "reserved for internal use"},
		{"source reserved", "source", true, "reserved for internal use"},
		{"model-output allowed", "model-output", false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			job := &NotebookValidationJob{
				Spec: NotebookValidationJobSpec{
					PodConfig: PodConfigSpec{
						Volumes: []Volume{
							{
								Name: tc.volumeName,
								EmptyDir: &EmptyDirVolumeSource{
									SizeLimit: "1Gi",
								},
							},
						},
						VolumeMounts: []VolumeMount{
							{
								Name:      tc.volumeName,
								MountPath: "/test",
							},
						},
					},
				},
			}

			err := validateVolumes(job)

			if tc.expectError && err == nil {
				t.Errorf("Expected error for reserved volume name %q, got nil", tc.volumeName)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for volume name %q, got: %v", tc.volumeName, err)
			}
			if tc.expectError && err != nil && tc.errorContain != "" {
				if !containsString(err.Error(), tc.errorContain) {
					t.Errorf("Expected error to contain %q, got: %v", tc.errorContain, err)
				}
			}
		})
	}
}

// TestValidateVolumes_DuplicateVolumeName tests that duplicate volume names are rejected
func TestValidateVolumes_DuplicateVolumeName(t *testing.T) {
	job := &NotebookValidationJob{
		Spec: NotebookValidationJobSpec{
			PodConfig: PodConfigSpec{
				Volumes: []Volume{
					{
						Name: "data",
						EmptyDir: &EmptyDirVolumeSource{
							SizeLimit: "1Gi",
						},
					},
					{
						Name: "data", // Duplicate!
						EmptyDir: &EmptyDirVolumeSource{
							SizeLimit: "2Gi",
						},
					},
				},
				VolumeMounts: []VolumeMount{
					{
						Name:      "data",
						MountPath: "/data",
					},
				},
			},
		},
	}

	err := validateVolumes(job)
	if err == nil {
		t.Error("Expected error for duplicate volume name, got nil")
	}
	if err != nil && !containsString(err.Error(), "duplicate volume name") {
		t.Errorf("Expected error to mention 'duplicate volume name', got: %v", err)
	}
}

// TestValidateVolumes_NoVolumeSource tests that volumes without a source are rejected
func TestValidateVolumes_NoVolumeSource(t *testing.T) {
	job := &NotebookValidationJob{
		Spec: NotebookValidationJobSpec{
			PodConfig: PodConfigSpec{
				Volumes: []Volume{
					{
						Name: "empty",
						// No source specified!
					},
				},
				VolumeMounts: []VolumeMount{},
			},
		},
	}

	err := validateVolumes(job)
	if err == nil {
		t.Error("Expected error for volume without source, got nil")
	}
	if err != nil && !containsString(err.Error(), "must specify exactly one volume source") {
		t.Errorf("Expected error to mention 'must specify exactly one volume source', got: %v", err)
	}
}

// TestValidateVolumes_MultipleVolumeSources tests that volumes with multiple sources are rejected
func TestValidateVolumes_MultipleVolumeSources(t *testing.T) {
	job := &NotebookValidationJob{
		Spec: NotebookValidationJobSpec{
			PodConfig: PodConfigSpec{
				Volumes: []Volume{
					{
						Name: "confused",
						PersistentVolumeClaim: &PersistentVolumeClaimVolumeSource{
							ClaimName: "my-pvc",
						},
						EmptyDir: &EmptyDirVolumeSource{
							SizeLimit: "1Gi",
						},
						// Two sources specified!
					},
				},
				VolumeMounts: []VolumeMount{},
			},
		},
	}

	err := validateVolumes(job)
	if err == nil {
		t.Error("Expected error for volume with multiple sources, got nil")
	}
	if err != nil && !containsString(err.Error(), "specifies multiple volume sources") {
		t.Errorf("Expected error to mention 'specifies multiple volume sources', got: %v", err)
	}
}

// TestValidateVolumes_UndefinedVolumeMount tests that volume mounts referencing undefined volumes are rejected
func TestValidateVolumes_UndefinedVolumeMount(t *testing.T) {
	job := &NotebookValidationJob{
		Spec: NotebookValidationJobSpec{
			PodConfig: PodConfigSpec{
				Volumes: []Volume{
					{
						Name: "data",
						EmptyDir: &EmptyDirVolumeSource{
							SizeLimit: "1Gi",
						},
					},
				},
				VolumeMounts: []VolumeMount{
					{
						Name:      "nonexistent", // References undefined volume!
						MountPath: "/mnt",
					},
				},
			},
		},
	}

	err := validateVolumes(job)
	if err == nil {
		t.Error("Expected error for volume mount referencing undefined volume, got nil")
	}
	if err != nil && !containsString(err.Error(), "undefined volume") {
		t.Errorf("Expected error to mention 'undefined volume', got: %v", err)
	}
}

// TestValidateVolumes_DuplicateMountPath tests that duplicate mount paths are rejected
func TestValidateVolumes_DuplicateMountPath(t *testing.T) {
	job := &NotebookValidationJob{
		Spec: NotebookValidationJobSpec{
			PodConfig: PodConfigSpec{
				Volumes: []Volume{
					{
						Name: "vol1",
						EmptyDir: &EmptyDirVolumeSource{
							SizeLimit: "1Gi",
						},
					},
					{
						Name: "vol2",
						EmptyDir: &EmptyDirVolumeSource{
							SizeLimit: "2Gi",
						},
					},
				},
				VolumeMounts: []VolumeMount{
					{
						Name:      "vol1",
						MountPath: "/data",
					},
					{
						Name:      "vol2",
						MountPath: "/data", // Duplicate mount path!
					},
				},
			},
		},
	}

	err := validateVolumes(job)
	if err == nil {
		t.Error("Expected error for duplicate mount path, got nil")
	}
	if err != nil && !containsString(err.Error(), "duplicate mount path") {
		t.Errorf("Expected error to mention 'duplicate mount path', got: %v", err)
	}
}

// TestValidateVolumes_MultipleValidVolumes tests a valid configuration with multiple volumes
func TestValidateVolumes_MultipleValidVolumes(t *testing.T) {
	job := &NotebookValidationJob{
		Spec: NotebookValidationJobSpec{
			PodConfig: PodConfigSpec{
				Volumes: []Volume{
					{
						Name: "model-output",
						PersistentVolumeClaim: &PersistentVolumeClaimVolumeSource{
							ClaimName: "trained-models-pvc",
						},
					},
					{
						Name: "training-data",
						PersistentVolumeClaim: &PersistentVolumeClaimVolumeSource{
							ClaimName: "datasets-pvc",
							ReadOnly:  true,
						},
					},
					{
						Name: "config",
						ConfigMap: &ConfigMapVolumeSource{
							Name: "notebook-config",
						},
					},
					{
						Name: "scratch",
						EmptyDir: &EmptyDirVolumeSource{
							Medium:    "Memory",
							SizeLimit: "1Gi",
						},
					},
				},
				VolumeMounts: []VolumeMount{
					{
						Name:      "model-output",
						MountPath: "/models",
					},
					{
						Name:      "training-data",
						MountPath: "/data",
						ReadOnly:  true,
					},
					{
						Name:      "config",
						MountPath: "/config",
					},
					{
						Name:      "scratch",
						MountPath: "/tmp/scratch",
					},
				},
			},
		},
	}

	err := validateVolumes(job)
	if err != nil {
		t.Errorf("Expected no error for valid multiple volumes, got: %v", err)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
