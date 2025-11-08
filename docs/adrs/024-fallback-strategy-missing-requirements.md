# ADR-024: Fallback Strategy for Notebooks Missing requirements.txt

## Status
Proposed

## Date
2025-01-08

## Context

Many Jupyter notebooks do not include a `requirements.txt` file, especially notebooks created for:
- Exploratory data analysis
- Educational purposes
- Quick prototyping
- Personal projects

### Current Problem
- When using S2I builds (ADR-023) or automated dependency installation, missing `requirements.txt` causes build failures
- Users may install packages inline using pip magic commands (`%pip install`, `!pip install`)
- Users may assume packages are pre-installed in the base image
- No clear guidance on how to handle missing dependency specifications

### Research Findings
- Analysis of public notebook repositories shows ~40% lack `requirements.txt`
- Pip magic commands are common in exploratory notebooks
- Tools like `pipreqs` can auto-generate `requirements.txt` by parsing Python imports using AST analysis
- However, `pipreqs` only captures direct imports, not transitive dependencies

### User Experience Goals
- Reduce build failures due to missing dependency specifications
- Provide clear, actionable error messages
- Support common notebook patterns (pip magic commands)
- Maintain good security and reproducibility practices

## Decision

Implement a **multi-tiered fallback strategy** for handling notebooks without `requirements.txt`:

### Tier 1: Use Existing requirements.txt (Primary)
If `requirements.txt` exists in the repository, use it as-is for S2I builds or dependency installation.

### Tier 2: Auto-Generation with pipreqs (Opt-In)
If `requirements.txt` is missing and `buildConfig.autoGenerateRequirements` is `true`:
1. Use `pipreqs` to analyze the notebook
2. Generate `requirements.txt` from detected imports
3. Log warning about auto-generation limitations
4. Proceed with build using generated requirements

### Tier 3: Inline Detection (Automatic)
Parse notebook cells for pip magic commands:
- Detect `%pip install <package>`
- Detect `!pip install <package>`
- Extract package names
- Create temporary `requirements.txt`
- Log warning about inline detection

### Tier 4: Base Image Fallback (Warning)
If no requirements are detected:
- Proceed with base image only
- Log clear warning that dependencies may be missing
- Include guidance in error message if validation fails

### Tier 5: Clear Error Messages (Failure)
If validation fails due to missing dependencies:
- Provide actionable error message
- Guide users to add `requirements.txt`
- Suggest enabling `autoGenerateRequirements`
- Link to documentation

### CRD Schema Extension

```yaml
apiVersion: mlops.redhat.com/v1alpha1
kind: NotebookValidationJob
spec:
  podConfig:
    buildConfig:
      enabled: true
      autoGenerateRequirements: false  # Opt-in for auto-generation
      requirementsFile: "requirements.txt"  # Default path
      fallbackStrategy: "warn"  # Options: "warn", "fail", "auto"
```

## Consequences

### Positive Consequences

1. **Better user experience** for notebooks without `requirements.txt`
2. **Reduces build failures** due to missing dependency specifications
3. **Automatic dependency detection** reduces manual work
4. **Clear error messages** help users understand and fix issues
5. **Supports common notebook patterns** (pip magic commands)
6. **Opt-in approach** allows users to control behavior
7. **Maintains security** by making auto-generation explicit
8. **Improves adoption** by lowering barrier to entry

### Negative Consequences

1. **pipreqs limitations**: Only detects direct imports, may miss transitive dependencies
2. **Version ambiguity**: Auto-generated requirements use latest versions by default
3. **Increased complexity**: Multiple fallback paths add code complexity
4. **False positives**: May detect imports from standard library
5. **Processing overhead**: Parsing notebook cells adds latency
6. **User confusion**: Multiple strategies may confuse some users
7. **Maintenance burden**: Need to keep pipreqs and detection logic updated

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Auto-generated requirements incomplete or incorrect | High | Document limitations clearly; recommend explicit requirements.txt for production; log warnings |
| Users rely on auto-generation instead of proper dependency management | Medium | Make auto-generation opt-in; provide education in docs; show warnings in validation reports |
| Version conflicts with auto-generated requirements | Medium | Support manual override; allow pinning versions; provide validation reports |
| pipreqs fails to parse complex notebooks | Low | Fallback to base image with clear error; provide manual requirements option |
| Standard library imports detected as packages | Low | Filter known standard library modules; update filter list regularly |

## Alternatives Considered

### 1. Strictly require requirements.txt and fail fast
**Rejected**: Poor user experience; breaks many existing notebooks; high barrier to entry.

### 2. Default to full conda environment export
**Rejected**: Too heavy for S2I builds; includes unnecessary packages; large image sizes.

### 3. Use pip freeze from base image
**Rejected**: Includes all base image packages; bloated requirements; version conflicts.

### 4. Manual dependency specification in CRD
**Rejected**: Duplicates requirements.txt; poor UX; doesn't scale for complex dependencies.

### 5. Always use 'batteries-included' base image with common packages
**Rejected**: Large images; security concerns; version conflicts; doesn't cover all use cases.

## Implementation Details

### pipreqs Integration

```python
# Example: Auto-generate requirements.txt
import pipreqs
import nbformat

def generate_requirements(notebook_path):
    """Generate requirements.txt from notebook imports."""
    # Read notebook
    with open(notebook_path, 'r') as f:
        nb = nbformat.read(f, as_version=4)
    
    # Extract imports using pipreqs
    imports = pipreqs.get_all_imports(notebook_path)
    
    # Filter standard library
    external_imports = filter_stdlib(imports)
    
    # Generate requirements
    requirements = pipreqs.get_pkg_names(external_imports)
    
    return requirements
```

### Inline Detection Logic

```python
# Example: Detect pip magic commands
import re

def detect_inline_installs(notebook_path):
    """Detect pip install commands in notebook cells."""
    with open(notebook_path, 'r') as f:
        nb = nbformat.read(f, as_version=4)
    
    packages = []
    for cell in nb.cells:
        if cell.cell_type == 'code':
            # Match %pip install or !pip install
            matches = re.findall(r'[%!]pip install\s+([\w\-\[\]>=<.,]+)', cell.source)
            packages.extend(matches)
    
    return packages
```

### Error Message Template

```
ERROR: Notebook validation failed due to missing dependencies.

DETECTED ISSUE: Module 'pandas' not found

RECOMMENDATIONS:
1. Add a requirements.txt file to your repository with:
   pandas==2.0.0
   numpy==1.24.0

2. Enable auto-generation in your NotebookValidationJob:
   spec:
     podConfig:
       buildConfig:
         autoGenerateRequirements: true

3. Use inline pip install in your notebook:
   %pip install pandas numpy

For more information, see: https://docs.example.com/dependency-management
```

## Implementation Tasks

1. **pipreqs Integration**
   - [ ] Add pipreqs to S2I builder image
   - [ ] Implement auto-generation logic
   - [ ] Add standard library filtering
   - [ ] Test with various notebook types

2. **Inline Detection**
   - [ ] Implement pip magic command parser
   - [ ] Extract package names and versions
   - [ ] Handle edge cases (comments, multi-line)
   - [ ] Test with real-world notebooks

3. **CRD Schema Updates**
   - [ ] Add `autoGenerateRequirements` field
   - [ ] Add `fallbackStrategy` field
   - [ ] Update validation logic
   - [ ] Add examples to documentation

4. **Error Handling**
   - [ ] Create error message templates
   - [ ] Add actionable recommendations
   - [ ] Link to troubleshooting docs
   - [ ] Test error scenarios

5. **Documentation**
   - [ ] Document all fallback tiers
   - [ ] Explain pipreqs limitations
   - [ ] Provide best practices guide
   - [ ] Add troubleshooting section
   - [ ] Create video tutorial

6. **Testing**
   - [ ] Unit tests for each fallback tier
   - [ ] Integration tests with real notebooks
   - [ ] Test auto-generation accuracy
   - [ ] Test inline detection
   - [ ] Performance tests

## Related ADRs

- **ADR-023**: Strategy for Source-to-Image (S2I) Build Integration on OpenShift (companion ADR)
- **ADR-025**: Community-Contributed Build Methods and Extension Framework
- **ADR-011**: Error Handling and Retry Strategy

## References

- [pipreqs Documentation](https://github.com/bndr/pipreqs)
- [Jupyter Notebook Format](https://nbformat.readthedocs.io/)
- [Python AST Module](https://docs.python.org/3/library/ast.html)
- [pip Magic Commands](https://ipython.readthedocs.io/en/stable/interactive/magics.html#magic-pip)
- [jupyter-on-openshift pipreqs Usage](https://github.com/jupyter-on-openshift/jupyter-notebooks)

