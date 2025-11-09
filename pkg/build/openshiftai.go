package build

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	imagev1 "github.com/openshift/api/image/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// OpenShiftAINamespace is the namespace where OpenShift AI ImageStreams are stored
	OpenShiftAINamespace = "redhat-ods-applications"

	// Annotation keys used by OpenShift AI
	AnnotationImageName  = "opendatahub.io/notebook-image-name"
	AnnotationImageDesc  = "opendatahub.io/notebook-image-desc"
	AnnotationImageOrder = "opendatahub.io/notebook-image-order"
	AnnotationImageURL   = "opendatahub.io/notebook-image-url"
)

// ImageStreamInfo contains information about an OpenShift AI ImageStream
type ImageStreamInfo struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description"`
	Order       int      `json:"order"`
	Tags        []string `json:"tags"`
	LatestTag   string   `json:"latestTag"`
	ImageRef    string   `json:"imageRef"`
	S2IEnabled  bool     `json:"s2iEnabled"`
}

// OpenShiftAIHelper provides utilities for working with OpenShift AI ImageStreams
type OpenShiftAIHelper struct {
	client client.Client
}

// NewOpenShiftAIHelper creates a new OpenShiftAI helper
func NewOpenShiftAIHelper(c client.Client) *OpenShiftAIHelper {
	return &OpenShiftAIHelper{client: c}
}

// IsInstalled checks if OpenShift AI is installed by checking if the namespace exists
func (h *OpenShiftAIHelper) IsInstalled(ctx context.Context) bool {
	logger := log.FromContext(ctx)

	// Try to list ImageStreams in the OpenShift AI namespace
	imageStreamList := &imagev1.ImageStreamList{}
	err := h.client.List(ctx, imageStreamList, client.InNamespace(OpenShiftAINamespace))
	if err != nil {
		if errors.IsNotFound(err) || errors.IsForbidden(err) {
			logger.V(1).Info("OpenShift AI not installed or not accessible", "namespace", OpenShiftAINamespace)
			return false
		}
		logger.Error(err, "Error checking for OpenShift AI installation")
		return false
	}

	logger.Info("OpenShift AI detected", "namespace", OpenShiftAINamespace, "imageStreamCount", len(imageStreamList.Items))
	return true
}

// ListS2IImageStreams lists all S2I-enabled ImageStreams in OpenShift AI
func (h *OpenShiftAIHelper) ListS2IImageStreams(ctx context.Context) ([]ImageStreamInfo, error) {
	logger := log.FromContext(ctx)

	imageStreamList := &imagev1.ImageStreamList{}
	err := h.client.List(ctx, imageStreamList, client.InNamespace(OpenShiftAINamespace))
	if err != nil {
		return nil, fmt.Errorf("failed to list ImageStreams: %w", err)
	}

	var s2iImages []ImageStreamInfo
	for _, is := range imageStreamList.Items {
		// Only include S2I-enabled images (those starting with "s2i-")
		if len(is.Name) < 4 || is.Name[:4] != "s2i-" {
			continue
		}

		info := h.parseImageStream(&is)
		s2iImages = append(s2iImages, info)
	}

	// Sort by order annotation
	sort.Slice(s2iImages, func(i, j int) bool {
		return s2iImages[i].Order < s2iImages[j].Order
	})

	logger.Info("Found S2I-enabled ImageStreams", "count", len(s2iImages))
	return s2iImages, nil
}

// ListAllImageStreams lists all ImageStreams in OpenShift AI (including non-S2I)
func (h *OpenShiftAIHelper) ListAllImageStreams(ctx context.Context) ([]ImageStreamInfo, error) {
	logger := log.FromContext(ctx)

	imageStreamList := &imagev1.ImageStreamList{}
	err := h.client.List(ctx, imageStreamList, client.InNamespace(OpenShiftAINamespace))
	if err != nil {
		return nil, fmt.Errorf("failed to list ImageStreams: %w", err)
	}

	var images []ImageStreamInfo
	for _, is := range imageStreamList.Items {
		info := h.parseImageStream(&is)
		images = append(images, info)
	}

	// Sort by order annotation
	sort.Slice(images, func(i, j int) bool {
		return images[i].Order < images[j].Order
	})

	logger.Info("Found ImageStreams", "count", len(images))
	return images, nil
}

// GetRecommendedS2IImage returns the recommended S2I image for notebook validation
func (h *OpenShiftAIHelper) GetRecommendedS2IImage(ctx context.Context) (*ImageStreamInfo, error) {
	s2iImages, err := h.ListS2IImageStreams(ctx)
	if err != nil {
		return nil, err
	}

	if len(s2iImages) == 0 {
		return nil, fmt.Errorf("no S2I-enabled ImageStreams found in OpenShift AI")
	}

	// Return the first one (lowest order number) which is typically s2i-minimal-notebook
	return &s2iImages[0], nil
}

// GetImageStreamByName retrieves a specific ImageStream by name
func (h *OpenShiftAIHelper) GetImageStreamByName(ctx context.Context, name string) (*ImageStreamInfo, error) {
	is := &imagev1.ImageStream{}
	err := h.client.Get(ctx, client.ObjectKey{
		Namespace: OpenShiftAINamespace,
		Name:      name,
	}, is)
	if err != nil {
		return nil, fmt.Errorf("failed to get ImageStream %s: %w", name, err)
	}

	info := h.parseImageStream(is)
	return &info, nil
}

// parseImageStream extracts information from an ImageStream
func (h *OpenShiftAIHelper) parseImageStream(is *imagev1.ImageStream) ImageStreamInfo {
	info := ImageStreamInfo{
		Name:       is.Name,
		S2IEnabled: len(is.Name) >= 4 && is.Name[:4] == "s2i-",
	}

	// Extract annotations
	if is.Annotations != nil {
		info.DisplayName = is.Annotations[AnnotationImageName]
		info.Description = is.Annotations[AnnotationImageDesc]

		// Parse order
		if orderStr, ok := is.Annotations[AnnotationImageOrder]; ok {
			if order, err := strconv.Atoi(orderStr); err == nil {
				info.Order = order
			}
		}
	}

	// Extract tags
	if is.Spec.Tags != nil {
		for _, tag := range is.Spec.Tags {
			info.Tags = append(info.Tags, tag.Name)
		}
	}

	// Determine latest tag (prefer "2025.1", then "latest", then first available)
	if len(info.Tags) > 0 {
		for _, tag := range info.Tags {
			if tag == "2025.1" {
				info.LatestTag = tag
				break
			}
		}
		if info.LatestTag == "" {
			for _, tag := range info.Tags {
				if tag == "latest" {
					info.LatestTag = tag
					break
				}
			}
		}
		if info.LatestTag == "" {
			// Use the last tag in the list (usually the newest)
			info.LatestTag = info.Tags[len(info.Tags)-1]
		}
	}

	// Build image reference
	if info.LatestTag != "" {
		info.ImageRef = fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%s/%s:%s",
			OpenShiftAINamespace, is.Name, info.LatestTag)
	}

	return info
}

// FormatImageStreamList formats a list of ImageStreams for display
func FormatImageStreamList(images []ImageStreamInfo) string {
	if len(images) == 0 {
		return "No ImageStreams found"
	}

	result := "Available OpenShift AI ImageStreams:\n\n"
	for _, img := range images {
		s2iIndicator := ""
		if img.S2IEnabled {
			s2iIndicator = " [S2I]"
		}
		result += fmt.Sprintf("  %d. %s%s\n", img.Order, img.DisplayName, s2iIndicator)
		result += fmt.Sprintf("     Name: %s\n", img.Name)
		result += fmt.Sprintf("     Description: %s\n", img.Description)
		result += fmt.Sprintf("     Tags: %v\n", img.Tags)
		result += fmt.Sprintf("     Recommended: %s\n", img.ImageRef)
		result += "\n"
	}

	return result
}
