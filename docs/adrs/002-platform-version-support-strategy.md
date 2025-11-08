# ADR 002: Platform Version Support Strategy

## Status
Accepted

## Context

The Jupyter Notebook Validator Operator must run reliably across multiple Kubernetes and OpenShift versions. Our target users include:
- **OpenShift Users**: Organizations running OpenShift 4.18, 4.19, and 4.20
- **Kubernetes Users**: Community users running upstream Kubernetes clusters
- **Enterprise Customers**: Organizations with strict version requirements and long support cycles

### Current Environment
- **Initial Target**: OpenShift 4.18 (as specified in PRD)
- **Future Targets**: OpenShift 4.19, 4.20, and latest stable Kubernetes
- **Deployment Context**: Production MLOps workflows requiring stability and predictability

### Technical Challenges
1. **API Deprecations**: Kubernetes APIs evolve, with deprecations and removals across versions
2. **Feature Availability**: Different versions support different features (e.g., CRD conversion webhooks, admission webhooks)
3. **Testing Complexity**: Each supported version requires dedicated test infrastructure
4. **Maintenance Burden**: Backporting fixes and features across multiple versions
5. **User Expectations**: Enterprise users expect long-term support; community users want latest features

### Version Landscape

#### OpenShift Versions
| Version | Kubernetes Equivalent | Release Date | Support End |
|---------|----------------------|--------------|-------------|
| 4.18    | 1.31                 | Q4 2024      | ~Q4 2026    |
| 4.19    | 1.32                 | Q1 2025      | ~Q1 2027    |
| 4.20    | 1.33                 | Q2 2025      | ~Q2 2027    |

#### Kubernetes Versions
| Version | Release Date | Support End |
|---------|--------------|-------------|
| 1.25    | Aug 2022     | Oct 2023    |
| 1.26    | Dec 2022     | Feb 2024    |
| 1.27    | Apr 2023     | Jun 2024    |
| 1.28    | Aug 2023     | Oct 2024    |
| 1.29    | Dec 2023     | Feb 2025    |
| 1.30    | Apr 2024     | Jun 2025    |
| 1.31    | Aug 2024     | Oct 2025    |

### Available Options

#### Option 1: Latest Only
- **Support**: Only the latest OpenShift and Kubernetes versions
- **Pros**: Simplest to maintain, fastest feature adoption, minimal testing overhead
- **Cons**: Excludes many enterprise users, forces frequent upgrades, limits adoption

#### Option 2: Rolling Window (N-2)
- **Support**: Current version and two previous versions
- **Pros**: Balances support breadth with maintenance burden, industry standard
- **Cons**: Still requires multi-version testing, may exclude some users

#### Option 3: Long-Term Support (LTS)
- **Support**: Specific versions for extended periods (e.g., 2+ years)
- **Pros**: Predictable for enterprise users, aligns with OpenShift support model
- **Cons**: High maintenance burden, slower feature adoption, complex backporting

#### Option 4: Hybrid Approach
- **Support**: OpenShift 4.18-4.20 + Kubernetes 1.25+
- **Pros**: Covers both enterprise (OpenShift) and community (K8s) users, manageable scope
- **Cons**: Requires careful API compatibility management, moderate testing complexity

## Decision

We will adopt a **Hybrid Support Strategy**:

### Supported Versions
- **OpenShift**: 4.18, 4.19, 4.20 (explicit support)
- **Kubernetes**: 1.25+ (best-effort support for upstream)

### Support Tiers

#### Tier 1: Certified Support (OpenShift 4.18-4.20)
- **Testing**: Dedicated e2e test suite per version in CI/CD
- **Guarantees**: Full compatibility, bug fixes, security patches
- **Documentation**: Version-specific installation and troubleshooting guides
- **Support**: Official support channels, SLAs for enterprise customers

#### Tier 2: Community Support (Kubernetes 1.25+)
- **Testing**: Automated tests against latest stable Kubernetes in CI/CD
- **Guarantees**: Best-effort compatibility, community-driven bug fixes
- **Documentation**: General installation guide with version notes
- **Support**: Community forums, GitHub issues

### Version Deprecation Policy
1. **Announcement**: Deprecation announced 6 months before support end
2. **Grace Period**: 3 months of security-only patches after deprecation
3. **End of Life**: Version removed from test matrix and documentation

### API Compatibility Strategy
- Use only Kubernetes APIs available in all supported versions
- Implement feature detection for version-specific capabilities
- Maintain API compatibility matrix in documentation

## Consequences

### Positive
- **Broad Adoption**: Covers both enterprise (OpenShift) and community (Kubernetes) users
- **Predictable Support**: Clear support tiers and deprecation policy
- **Manageable Scope**: Limited to 3 OpenShift versions + latest K8s
- **Enterprise-Friendly**: Aligns with OpenShift's support lifecycle
- **Community-Friendly**: Supports latest Kubernetes features

### Negative
- **Testing Complexity**: Requires CI/CD infrastructure for multiple cluster versions
- **Maintenance Burden**: Must track API deprecations across versions
- **Documentation Overhead**: Version-specific guides and compatibility matrices
- **Backporting Effort**: May need to backport critical fixes to older versions

### Neutral
- **API Constraints**: Must use lowest-common-denominator APIs across versions
- **Feature Gating**: New features may require version checks or feature flags

## Implementation Notes

### CI/CD Test Matrix
```yaml
# .github/workflows/e2e-tests.yml
strategy:
  matrix:
    platform:
      - openshift-4.18
      - openshift-4.19
      - openshift-4.20
      - kubernetes-1.25
      - kubernetes-latest
```

### Version Detection
```go
// pkg/version/detector.go
func DetectPlatformVersion(client kubernetes.Interface) (*PlatformInfo, error) {
    version, err := client.Discovery().ServerVersion()
    if err != nil {
        return nil, err
    }
    
    return &PlatformInfo{
        Major:      version.Major,
        Minor:      version.Minor,
        GitVersion: version.GitVersion,
        Platform:   detectPlatform(version),
    }, nil
}
```

### API Compatibility Matrix
| Feature                  | K8s 1.25 | K8s 1.26+ | OCP 4.18 | OCP 4.19+ |
|--------------------------|----------|-----------|----------|-----------|
| CRD v1                   | ✅       | ✅        | ✅       | ✅        |
| Conversion Webhooks      | ✅       | ✅        | ✅       | ✅        |
| Server-Side Apply        | ✅       | ✅        | ✅       | ✅        |
| Pod Security Admission   | ✅       | ✅        | ✅       | ✅        |

### Testing Infrastructure
```bash
# Test against OpenShift 4.18
make test-e2e PLATFORM=openshift VERSION=4.18

# Test against latest Kubernetes
make test-e2e PLATFORM=kubernetes VERSION=latest
```

### Documentation Structure
```
docs/
├── installation/
│   ├── openshift-4.18.md
│   ├── openshift-4.19.md
│   ├── openshift-4.20.md
│   └── kubernetes.md
├── compatibility-matrix.md
└── version-support-policy.md
```

### Upgrade Path
1. **Monitor Releases**: Track OpenShift and Kubernetes release schedules
2. **Early Testing**: Test against beta/RC versions before GA
3. **Deprecation Notices**: Announce deprecations in release notes and documentation
4. **Migration Guides**: Provide upgrade guides for users on deprecated versions

## References

- [Kubernetes Version Skew Policy](https://kubernetes.io/releases/version-skew-policy/)
- [OpenShift Life Cycle Policy](https://access.redhat.com/support/policy/updates/openshift)
- [Kubernetes API Deprecation Policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/)
- [Operator SDK Version Compatibility](https://sdk.operatorframework.io/docs/upgrading-sdk-version/)

## Related ADRs

- ADR 001: Operator Framework and SDK Version
- ADR 003: CRD Schema Design & Versioning
- ADR 007: Testing & Validation Strategy

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-07 | Team   | Initial decision |

