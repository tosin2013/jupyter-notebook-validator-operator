# ADR-046: Multi-Version Bundle Strategy and Upgrade Chain

**Status**: Accepted (Superseded by ADR-047 for versioning fix)  
**Date**: 2025-12-03  
**Deciders**: Tosin Akinosho, Community Operators Maintainer Feedback  
**Context**: Operator Hub Bundle Submission, Multi-OpenShift Version Support

## Context and Problem Statement

The Jupyter Notebook Validator Operator needs to support multiple OpenShift versions (4.18, 4.19, 4.20) with version-specific features and optimizations. We need a versioning strategy that:

1. Supports multiple OpenShift versions simultaneously
2. Provides clear upgrade paths for users
3. Complies with Operator Hub requirements
4. Enables version-specific features (e.g., Tekton v1 API in 4.20)
5. Maintains separate release branches for each OpenShift version

## Decision Drivers

1. **Operator Hub Requirements**: Consecutive upgrade chain required (no conflicts)
2. **Multi-Version Support**: Need to support OpenShift 4.18, 4.19, 4.20
3. **Version-Specific Features**: Leverage platform-specific capabilities
4. **Branch Strategy**: Maintain separate release branches (release-4.18, release-4.19, release-4.20)
5. **User Experience**: Clear upgrade path and version mapping
6. **Automation Compatibility**: Must work with Operator Hub automation

## Decision Outcome

### **Consecutive Semantic Versioning Strategy**

Use different semantic versions for each OpenShift version to create a consecutive upgrade chain:

| Bundle Version | OpenShift Version | Release Branch | Operator Image Tag | Bundle Image Tag | Replaces |
|---------------|-------------------|----------------|-------------------|------------------|----------|
| **1.0.7** | 4.18+ | release-4.18 | `1.0.7-ocp4.18` | `1.0.7` | `1.0.3` |
| **1.0.8** | 4.19+ | release-4.19 | `1.0.8-ocp4.19` | `1.0.8` | `1.0.7` |
| **1.0.9** | 4.20+ | release-4.20 | `1.0.9-ocp4.20` | `1.0.9` | `1.0.8` |

### **Upgrade Chain**

```
1.0.3 → 1.0.7 (OCP 4.18) → 1.0.8 (OCP 4.19) → 1.0.9 (OCP 4.20)
```

### **Key Principles**

1. **Consecutive Versions**: Each bundle replaces the previous version (no conflicts)
2. **Semantic Versioning**: Bundle versions use pure semantic versioning (no OpenShift suffix)
3. **Image Tag Clarity**: Operator images include OpenShift version for clarity (`1.0.7-ocp4.18`)
4. **Bundle Tag Simplicity**: Bundle images use semantic version only (`1.0.7`)
5. **Branch Isolation**: Each OpenShift version has its own release branch
6. **Forward Compatibility**: Newer versions support older OpenShift versions (e.g., 1.0.9 works on 4.18+)

## Implementation Guidelines for Future Developers

### **When Adding Support for a New OpenShift Version**

**Example: Adding OpenShift 4.21 Support**

1. **Create Release Branch**
   ```bash
   git checkout -b release-4.21 main
   ```

2. **Determine Next Version**
   - Check latest bundle version (e.g., 1.0.9 for 4.20)
   - Increment patch version: 1.0.9 → 1.0.10
   - Use minor version bump for major features: 1.0.9 → 1.1.0

3. **Update Makefile**
   ```makefile
   VERSION ?= 1.0.10
   ```

4. **Build Operator Image with OpenShift Tag**
   ```bash
   make docker-build IMG=quay.io/yourrepo/jupyter-notebook-validator-operator:1.0.10-ocp4.21
   make docker-push IMG=quay.io/yourrepo/jupyter-notebook-validator-operator:1.0.10-ocp4.21
   ```

5. **Generate Bundle with Semantic Version**
   ```bash
   make bundle IMG=quay.io/yourrepo/jupyter-notebook-validator-operator:1.0.10-ocp4.21
   ```

6. **Add Replaces Field to CSV**
   Edit `bundle/manifests/jupyter-notebook-validator-operator.clusterserviceversion.yaml`:
   ```yaml
   spec:
     replaces: jupyter-notebook-validator-operator.v1.0.9
   ```

7. **Build Bundle Image with Semantic Version Tag**
   ```bash
   make bundle-build BUNDLE_IMG=quay.io/yourrepo/jupyter-notebook-validator-operator-bundle:1.0.10
   make bundle-push BUNDLE_IMG=quay.io/yourrepo/jupyter-notebook-validator-operator-bundle:1.0.10
   ```

8. **Update Catalog**
   Add new bundle to `catalog/catalog.yaml`:
   ```yaml
   - name: jupyter-notebook-validator-operator.v1.0.10
     replaces: jupyter-notebook-validator-operator.v1.0.9
   ```

9. **Commit and Push**
   ```bash
   git add Makefile bundle/ config/
   git commit -m "build: Release version 1.0.10 for OpenShift 4.21"
   git push origin release-4.21
   ```

### **Version Numbering Guidelines**

**Patch Version Increment (1.0.X → 1.0.X+1)**:
- Bug fixes
- Minor feature additions
- OpenShift version support (if no major API changes)
- Dependency updates

**Minor Version Increment (1.X.0 → 1.X+1.0)**:
- New major features (e.g., GPU support, new build strategies)
- Significant API changes in dependencies (e.g., Tekton v1 → v2)
- Breaking changes in configuration

**Major Version Increment (X.0.0 → X+1.0.0)**:
- Breaking API changes in CRD
- Major architectural changes
- Incompatible upgrades

### **Testing Checklist**

Before releasing a new version:

- [ ] Operator builds successfully
- [ ] Bundle generates without errors
- [ ] CSV includes correct `replaces` field
- [ ] Bundle image tag uses semantic version only (no OpenShift suffix)
- [ ] Operator image tag includes OpenShift version for clarity
- [ ] Test deployment on target OpenShift version
- [ ] Test upgrade from previous version (if applicable)
- [ ] Validate catalog with `opm validate`
- [ ] Test tier1/tier2/tier3 notebooks
- [ ] Update documentation with version mapping

## Consequences

### Positive

- ✅ Consecutive upgrade chain (no conflicts)
- ✅ All bundles propagate to Operator Hub
- ✅ Clear upgrade path for users
- ✅ Compatible with Operator Hub automation
- ✅ Semantic versioning reflects feature additions
- ✅ Version-specific optimizations possible
- ✅ Branch isolation prevents cross-version conflicts

### Negative

- ⚠️ Need to rebuild bundles for each OpenShift version
- ⚠️ Version-to-OpenShift mapping requires documentation
- ⚠️ Multiple release branches to maintain
- ⚠️ Coordination needed for cross-version features

### Neutral

- 📝 Documentation must clearly map versions to OpenShift versions
- 📝 Release notes must explain version strategy
- 📝 Users must understand which version to install

## Related ADRs

- ADR-002: Platform Version Support Strategy
- ADR-006: Version Support Roadmap and Testing
- ADR-007: Distribution and Catalog Strategy
- ADR-045: Long-Term Strategic Deployment Plan
- ADR-047: Fix Bundle Versioning for Consecutive Upgrade Chain (implementation details)

## References

- OLM Bundle Documentation: https://olm.operatorframework.io/docs/concepts/olm-architecture/operator-catalog/creating-an-update-graph/
- Semantic Versioning: https://semver.org/
- Community Operators Guidelines: https://github.com/operator-framework/community-operators/blob/main/docs/contributing.md

