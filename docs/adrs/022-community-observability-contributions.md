# ADR 022: Community Observability Contributions

## Status
Proposed

## Context

The Jupyter Notebook Validator Operator provides built-in dashboards for core monitoring use cases (ADR-021). However, different organizations have unique observability needs:

1. **Multi-Cluster Monitoring**: Organizations running operators across multiple clusters
2. **Cost Optimization**: Teams focused on resource efficiency and cost reduction
3. **Security & Compliance**: Organizations with strict audit and compliance requirements
4. **Developer Experience**: Teams optimizing for developer productivity
5. **Custom Integrations**: Organizations with specific monitoring tools and workflows

### Current State

- **Built-In Dashboards**: 5 core dashboards maintained by the core team
- **Limited Coverage**: Cannot cover all use cases and monitoring tools
- **Community Interest**: Users requesting custom dashboards and integrations
- **Maintenance Burden**: Core team cannot maintain dashboards for all platforms

### Opportunity

Enable the community to contribute dashboards, alerts, and monitoring integrations while maintaining quality and consistency.

## Decision

We will establish a **community-driven observability contribution framework** that enables users to contribute dashboards, alerts, and monitoring integrations.

### Contribution Framework

**Tier 1: Built-In (Core Team)**
- 5 core OpenShift Console dashboards
- Basic Prometheus alerts
- ServiceMonitor configuration
- Maintained by core team

**Tier 2: Community-Contributed (Community)**
- Advanced dashboards (multi-cluster, cost, security, etc.)
- Platform-specific integrations (Datadog, New Relic, etc.)
- Custom alerts and runbooks
- Maintained by community contributors

### Community Dashboard Categories

We will accept contributions in these categories:

1. **Model-Aware Validation Dashboard** 🔴 NEEDS CONTRIBUTOR
   - Model health checks by platform
   - Prediction validation results
   - Platform detection metrics

2. **Multi-Cluster Dashboard** 🔴 NEEDS CONTRIBUTOR
   - Validation jobs across clusters
   - Cross-cluster success rates
   - Cluster-specific error patterns

3. **Cost Optimization Dashboard** 🔴 NEEDS CONTRIBUTOR
   - Resource requests vs. actual usage
   - Validation cost per notebook
   - Idle pod time analysis

4. **Security & Compliance Dashboard** 🔴 NEEDS CONTRIBUTOR
   - Credential usage patterns
   - Secret rotation status
   - RBAC violations
   - Audit log summary

5. **Developer Experience Dashboard** 🔴 NEEDS CONTRIBUTOR
   - Average validation time by user
   - Most common errors
   - Notebook complexity trends
   - User success rate

### Contribution Process

**Step 1: Proposal (GitHub Issue)**
- Create issue with `dashboard-proposal` label
- Describe use case and target audience
- List required metrics and queries
- Get community feedback

**Step 2: Implementation (Pull Request)**
- Create dashboard ConfigMap or Grafana JSON
- Add documentation in `docs/dashboards/<name>.md`
- Include screenshots and example queries
- Add tests (optional)

**Step 3: Review & Merge**
- Core team reviews for quality and security
- Community provides feedback
- Merge to `config/monitoring/community/` directory
- Add to community dashboard catalog

**Step 4: Maintenance**
- Contributor maintains dashboard
- Core team provides support and guidance
- Community can adopt orphaned dashboards

### Quality Standards

All community contributions must meet:

1. **Documentation**: Clear README with use case, installation, and screenshots
2. **Metrics**: Use only exposed operator metrics (no custom metrics)
3. **Security**: No sensitive data in queries or dashboards
4. **Testing**: Include example queries and expected results
5. **Licensing**: Apache 2.0 license

### Recognition Program

Contributors receive:
- **Badge**: "Dashboard Contributor" on GitHub profile
- **Newsletter Feature**: Highlighted in monthly newsletter
- **Speaking Opportunity**: Present dashboard at community call
- **Swag**: Contributor t-shirt and stickers

## Consequences

### Positive

1. **Extended Coverage**: Community can create dashboards for niche use cases
2. **Reduced Maintenance**: Core team focuses on core dashboards
3. **Innovation**: Community brings new ideas and approaches
4. **Engagement**: Increases community participation and ownership
5. **Ecosystem Growth**: Builds a rich ecosystem of monitoring tools

### Negative

1. **Quality Variance**: Community dashboards may vary in quality
2. **Maintenance Risk**: Contributors may abandon dashboards
3. **Support Burden**: Core team must provide guidance and support
4. **Fragmentation**: Too many dashboards can confuse users

### Mitigation

1. **Quality Guidelines**: Clear contribution guidelines and review process
2. **Adoption Process**: Community can adopt orphaned dashboards
3. **Support Channels**: Dedicated Slack channel and office hours
4. **Curation**: Maintain a curated list of recommended dashboards

## Implementation Plan

### Phase 1: Framework Setup (Week 1)
- [ ] Create `docs/COMMUNITY_OBSERVABILITY.md` guide
- [ ] Set up `config/monitoring/community/` directory
- [ ] Create dashboard proposal template
- [ ] Add contribution guidelines to CONTRIBUTING.md

### Phase 2: Community Outreach (Week 2)
- [ ] Create GitHub issues for 5 dashboard categories
- [ ] Announce on Slack and mailing list
- [ ] Host community call to explain process
- [ ] Create video tutorial on dashboard creation

### Phase 3: First Contributions (Week 3-4)
- [ ] Support first 2-3 community contributors
- [ ] Review and merge first community dashboards
- [ ] Document lessons learned
- [ ] Refine contribution process

### Phase 4: Recognition & Growth (Ongoing)
- [ ] Feature contributors in newsletter
- [ ] Host dashboard showcase at community call
- [ ] Send contributor swag
- [ ] Track and celebrate milestones

## Alternatives Considered

### Alternative 1: Core Team Maintains All Dashboards
**Rejected**: Unsustainable, limits innovation, high maintenance burden.

### Alternative 2: No Community Contributions
**Rejected**: Misses opportunity for community engagement and ecosystem growth.

### Alternative 3: Separate Repository for Dashboards
**Rejected**: Fragments the project, harder to discover, complicates installation.

## Related ADRs

- **ADR-010**: Observability and Monitoring Strategy (defines metrics)
- **ADR-020**: Model-Aware Validation Strategy (defines model metrics)
- **ADR-021**: OpenShift-Native Dashboard Strategy (defines built-in dashboards)

## References

- [Kubernetes SIG Contributor Experience](https://github.com/kubernetes/community/tree/master/sig-contributor-experience)
- [CNCF Project Contribution Guidelines](https://contribute.cncf.io/)
- [Prometheus Community Dashboards](https://grafana.com/grafana/dashboards/)
- [OpenShift Community](https://www.openshift.com/community)

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-08 | Team   | Initial community observability contribution framework |

